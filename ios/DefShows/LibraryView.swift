import SwiftUI

struct LibraryView: View {
    @EnvironmentObject private var session: Session
    @State private var shows: [TrackedShow] = []
    @State private var query = ""
    @State private var loading = false
    @State private var loaded = false
    @State private var error: String?

    private var filtered: [TrackedShow] {
        shows.filter { query.isEmpty || $0.show.title.localizedCaseInsensitiveContains(query) }
    }

    var body: some View {
        NavigationStack {
            List {
                if let error {
                    Section {
                        Text(error).foregroundStyle(.red)
                        Button("Повторить") { Task { await load() } }
                    }
                }
                if loading && !loaded {
                    ProgressView("Загружаем сериалы…")
                } else if loaded && shows.isEmpty {
                    ContentUnavailableView("Пока нет сериалов", systemImage: "tv", description: Text("Найди первый сериал во вкладке «Поиск» или «Подбор»."))
                } else if loaded && filtered.isEmpty {
                    ContentUnavailableView.search(text: query)
                }
                ForEach(filtered) { item in
                    NavigationLink {
                        ShowView(showID: item.id)
                    } label: {
                        HStack(spacing: 14) {
                            AsyncImage(url: session.posterURL(item.show.posterUrl)) { image in
                                image.resizable().scaledToFill()
                            } placeholder: {
                                Rectangle().fill(.quaternary).overlay { Image(systemName: "tv").foregroundStyle(.secondary) }
                            }
                            .frame(width: 62, height: 92).clipShape(RoundedRectangle(cornerRadius: 8))
                            VStack(alignment: .leading, spacing: 7) {
                                Text(item.show.title).font(.headline)
                                Text(item.userShow.statusTitle).font(.caption).foregroundStyle(.secondary)
                                ProgressView(value: Double(min(item.progress.watched, item.progress.total)), total: Double(max(item.progress.total, 1)))
                                Text("\(item.progress.watched) из \(item.progress.total) вышедших серий")
                                    .font(.caption).foregroundStyle(.secondary)
                            }
                            if item.userShow.favorite { Image(systemName: "heart.fill").foregroundStyle(.pink).accessibilityLabel("Избранное") }
                        }.padding(.vertical, 4)
                    }
                }
            }
            .navigationTitle("Мои сериалы")
            .searchable(text: $query, prompt: "Найти в коллекции")
            .refreshable { await load() }
            .onAppear { Task { await load() } }
            .onChange(of: session.collectionRevision) { _, _ in Task { await load() } }
        }
    }

    private func load() async {
        guard !loading else { return }
        loading = true
        defer { loading = false }
        do {
            let result: TrackedList = try await session.get("me/shows")
            shows = result.tracked
            loaded = true
            error = nil
        } catch is CancellationError { }
        catch { self.error = error.localizedDescription }
    }
}
