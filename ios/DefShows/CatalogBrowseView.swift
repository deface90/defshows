import SwiftUI

struct CatalogBrowseView: View {
    let discover: Bool
    @EnvironmentObject private var session: Session
    @State private var search = ""
    @State private var items: [CatalogItem] = []
    @State private var filters: DiscoveryFilters?
    @State private var genre = 0
    @State private var country = ""
    @State private var minimumRating = 0
    @State private var sort = "popularity.desc"
    @State private var page = 0
    @State private var totalPages = 0
    @State private var appliedQuery: [URLQueryItem] = []
    @State private var busy = false
    @State private var loaded = false
    @State private var error: String?

    var body: some View {
        List {
            if discover {
                Section("Подобрать сериал") {
                    Picker("Жанр", selection: $genre) {
                        Text("Любой").tag(0)
                        ForEach(filters?.genres ?? []) { Text($0.name).tag($0.id) }
                    }
                    Picker("Страна", selection: $country) {
                        Text("Любая").tag("")
                        ForEach(filters?.countries ?? []) { Text($0.name).tag($0.code) }
                    }
                    Picker("Рейтинг от", selection: $minimumRating) {
                        Text("Любой").tag(0)
                        ForEach(1..<10) { Text("\($0)").tag($0) }
                    }
                    Picker("Сортировка", selection: $sort) {
                        Text("Популярность").tag("popularity.desc")
                        Text("Рейтинг").tag("vote_average.desc")
                        Text("Новинки").tag("first_air_date.desc")
                    }
                    Button("Подобрать") { Task { await load(reset: true) } }
                }.disabled(busy)
            } else {
                Section {
                    HStack {
                        TextField("Название сериала", text: $search)
                            .submitLabel(.search).autocorrectionDisabled()
                            .onSubmit { Task { await load(reset: true) } }
                        Button { Task { await load(reset: true) } } label: {
                            Image(systemName: "magnifyingglass")
                        }.accessibilityLabel("Найти")
                    }
                }.disabled(busy)
            }
            if let error {
                Section {
                    Text(error).foregroundStyle(.red)
                    Button("Повторить") { Task { await load(reset: true) } }.disabled(busy)
                }
            }
            if busy { ProgressView("Загружаем…") }
            if !loaded && !discover && !busy && error == nil {
                ContentUnavailableView("Найди следующий сериал", systemImage: "magnifyingglass", description: Text("Поиск по каталогу TMDB."))
            } else if loaded && items.isEmpty && !busy {
                ContentUnavailableView("Ничего не найдено", systemImage: "magnifyingglass", description: Text("Попробуй другое название или фильтры."))
            }
            ForEach(items) { item in
                NavigationLink {
                    CatalogShowView(tmdbID: item.tmdbId)
                } label: {
                    CatalogRow(title: item.title, poster: item.posterUrl,
                               subtitle: item.voteAverage.map { String(format: "★ %.1f", $0) })
                }
            }
            if discover && page > 0 && page < min(totalPages, 500) {
                Button("Показать ещё") { Task { await load(reset: false) } }.disabled(busy)
            }
        }
        .navigationTitle(discover ? "Подбор" : "Поиск")
        .refreshable { await load(reset: true) }
        .task {
            guard discover, !loaded else { return }
            await load(reset: true)
        }
    }

    private func load(reset: Bool) async {
        guard !busy else { return }
        let term = search.trimmingCharacters(in: .whitespacesAndNewlines)
        guard discover || !term.isEmpty else { return }
        busy = true
        defer { busy = false }
        let nextPage = reset ? 1 : page + 1
        var query = appliedQuery
        if reset {
            query = discover ? [URLQueryItem(name: "sort", value: sort)] : [URLQueryItem(name: "q", value: term)]
            if discover {
                if genre != 0 { query.append(URLQueryItem(name: "genres", value: String(genre))) }
                if !country.isEmpty { query.append(URLQueryItem(name: "country", value: country)) }
                if minimumRating != 0 { query.append(URLQueryItem(name: "rating_min", value: String(minimumRating))) }
            }
        }
        do {
            if discover && filters == nil { filters = try await session.get("shows/discover/filters") }
            var pagedQuery = query
            if discover { pagedQuery.append(URLQueryItem(name: "page", value: String(nextPage))) }
            let result: CatalogResults = try await session.get(discover ? "shows/discover" : "shows/search", query: pagedQuery)
            try Task.checkCancellation()
            let existing = reset ? Set<Int>() : Set(items.map(\.id))
            items = (reset ? [] : items) + result.results.filter { !existing.contains($0.id) }
            appliedQuery = query
            page = nextPage
            totalPages = result.totalPages ?? 1
            loaded = true
            error = nil
        } catch is CancellationError { }
        catch { self.error = error.localizedDescription }
    }
}

struct CatalogShowView: View {
    let tmdbID: Int
    @EnvironmentObject private var session: Session
    @State private var detail: CatalogDetail?
    @State private var trackedID: Int?
    @State private var busy = false
    @State private var adding = false
    @State private var ready = false
    @State private var error: String?

    var body: some View {
        List {
            if let error {
                Text(error).foregroundStyle(.red)
                Button("Повторить") { Task { await load() } }.disabled(busy || adding)
            }
            if busy { ProgressView() }
            if let detail {
                ShowInformation(show: detail)
                Section {
                    if let trackedID {
                        NavigationLink("Открыть в моих сериалах") { ShowView(showID: trackedID) }
                    } else {
                        Button {
                            adding = true
                            Task {
                                defer { adding = false }
                                do {
                                    try await session.addShow(tmdbID: tmdbID)
                                    trackedID = detail.id
                                    error = nil
                                } catch { self.error = error.localizedDescription }
                            }
                        } label: { Label("Добавить в мои сериалы", systemImage: "plus.circle") }
                        .disabled(!ready || busy || adding)
                    }
                    if adding { ProgressView() }
                }
                ForEach((detail.seasons ?? []).newestFirst) { season in
                    Section(season.name.isEmpty ? "Сезон \(season.seasonNumber)" : season.name) {
                        Text("Серий: \(season.episodes.count)").font(.caption).foregroundStyle(.secondary)
                        if let rating = season.voteAverage, rating > 0 {
                            Text(String(format: "★ %.1f", rating)).font(.caption).foregroundStyle(.orange)
                        }
                        ForEach(season.episodes.sorted { $0.episodeNumber < $1.episodeNumber }) { episode in
                            EpisodeInformation(episode: episode)
                        }
                    }
                }
            }
        }
        .navigationTitle(detail?.title ?? "Сериал")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        guard !busy, !adding else { return }
        busy = true
        ready = false
        defer { busy = false }
        do {
            let show: CatalogDetail = try await session.get("shows/tmdb/\(tmdbID)")
            let collection: TrackedList = try await session.get("me/shows")
            detail = show
            trackedID = collection.tracked.first { $0.id == show.id }?.id
            ready = true
            error = nil
        } catch { self.error = error.localizedDescription }
    }
}
