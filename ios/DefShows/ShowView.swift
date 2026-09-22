import SwiftUI

struct ShowView: View {
    let showID: Int
    @EnvironmentObject private var session: Session
    @State private var detail: ShowDetail?
    @State private var tracked: TrackedShow?
    @State private var watched: Set<Int> = []
    @State private var expanded: Set<Int> = []
    @State private var loading = false
    @State private var changing = false
    @State private var confirmWatchAll = false
    @State private var error: String?

    private var nextEpisode: Episode? {
        guard let id = tracked?.progress.nextUnwatchedEpisodeId else { return nil }
        return detail?.seasons?.flatMap(\.episodes).first { $0.id == id }
    }

    var body: some View {
        List {
            if let error {
                Section {
                    Text(error).foregroundStyle(.red)
                    Button("Обновить") { Task { await load() } }.disabled(changing || loading)
                }
            }
            if let detail {
                ShowInformation(show: detail)
                if let tracked {
                    trackingSection(tracked)
                    Section("Продолжить просмотр") {
                        if let next = nextEpisode {
                            EpisodeInformation(episode: next)
                            Button("Отметить просмотренным") { Task { await toggle(next) } }
                                .disabled(changing || loading || !next.hasAired())
                        } else { Text("Все вышедшие серии просмотрены").foregroundStyle(.secondary) }
                    }
                    Section("Приватные заметки") {
                        NavigationLink {
                            ShowNotesView(showID: showID, seasons: detail.seasons ?? [])
                        } label: { Label("Мои заметки", systemImage: "note.text") }
                    }
                }
                if (detail.seasons ?? []).isEmpty {
                    ContentUnavailableView("Серии пока не загружены", systemImage: "tv")
                }
                ForEach((detail.seasons ?? []).newestFirst) { season in
                    seasonSection(season)
                }
            }
            if loading { ProgressView("Загружаем…") }
        }
        .navigationTitle(detail?.title ?? "Сериал")
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { await load() }
        .toolbar { if changing { ProgressView() } }
        .confirmationDialog("Отметить весь сериал просмотренным?", isPresented: $confirmWatchAll, titleVisibility: .visible) {
            Button("Отметить весь сериал") {
                Task { await perform { try await session.mutate("me/shows/\(showID)/watch") } }
            }
        } message: {
            Text("Будут отмечены все серии в каталоге, включая ещё не вышедшие. Статус изменится на «Просмотрено».")
        }
    }

    private func trackingSection(_ tracked: TrackedShow) -> some View {
        Section("Мой просмотр") {
            Menu {
                ForEach([("watching", "Смотрю"), ("plan_to_watch", "В планах"), ("on_hold", "На паузе"), ("completed", "Просмотрено"), ("dropped", "Брошено")], id: \.0) { value, title in
                    Button(title) {
                        Task { await perform { try await session.mutate("me/shows/\(showID)", method: "PATCH", body: ["status": value]) } }
                    }
                }
            } label: { LabeledContent("Мой статус", value: tracked.userShow.statusTitle) }
            .disabled(changing || loading)
            let progress = tracked.progress
            ProgressView(value: Double(min(progress.watched, progress.total)), total: Double(max(progress.total, 1)))
            Text("\(progress.watched) из \(progress.total) вышедших серий · \(progress.total > 0 ? Int((Double(progress.watched) / Double(progress.total) * 100).rounded()) : 0)%")
                .font(.caption).foregroundStyle(.secondary)
            Button("Сериал просмотрен целиком") { confirmWatchAll = true }
                .disabled(changing || loading)
        }
    }

    private func seasonSection(_ season: Season) -> some View {
        let aired = season.episodes.filter { $0.hasAired() }
        let count = aired.filter { watched.contains($0.id) }.count
        let complete = !aired.isEmpty && count == aired.count
        return Section {
            DisclosureGroup(isExpanded: Binding(
                get: { expanded.contains(season.id) },
                set: { if $0 { expanded.insert(season.id) } else { expanded.remove(season.id) } }
            )) {
                if tracked != nil {
                    Button(complete ? "Снять отметки сезона" : "Отметить вышедшие серии сезона") {
                        Task { await markSeason(season, watched: !complete) }
                    }.disabled(changing || loading || aired.isEmpty)
                }
                ForEach(season.episodes.sorted { $0.episodeNumber < $1.episodeNumber }) { episode in
                    HStack(spacing: 12) {
                        if tracked != nil {
                            Button { Task { await toggle(episode) } } label: {
                                Image(systemName: watched.contains(episode.id) ? "checkmark.circle.fill" : "circle")
                                    .font(.title2).frame(minWidth: 44, minHeight: 44)
                            }
                            .buttonStyle(.borderless)
                            .disabled(changing || loading || !episode.hasAired())
                            .accessibilityLabel("\(episode.code): \(watched.contains(episode.id) ? "снять отметку просмотра" : "отметить просмотренным")")
                        }
                        EpisodeInformation(episode: episode)
                    }
                }
            } label: {
                VStack(alignment: .leading, spacing: 4) {
                    Text(season.name.isEmpty ? "Сезон \(season.seasonNumber)" : season.name).font(.headline)
                    Text(tracked == nil ? "Серий: \(season.episodes.count)" : "\(count) из \(aired.count) вышедших серий\(complete ? " ✓" : "")")
                        .font(.caption).foregroundStyle(.secondary)
                    if let rating = season.voteAverage, rating > 0 {
                        Text(String(format: "★ %.1f", rating)).font(.caption).foregroundStyle(.orange)
                    }
                }
            }
        }
    }

    private func load() async {
        guard !loading, !changing else { return }
        loading = true
        defer { loading = false }
        do {
            let show: ShowDetail = try await session.get("shows/\(showID)")
            if detail == nil, let latest = (show.seasons ?? []).newestFirst.first { expanded.insert(latest.id) }
            detail = show
            try await reloadTracking()
            error = nil
        } catch is CancellationError { }
        catch { self.error = error.localizedDescription }
    }

    private func reloadTracking() async throws {
        let result: TrackedShow = try await session.get("me/shows/\(showID)")
        tracked = result
        watched = Set(result.progress.watchedEpisodeIds)
    }

    private func perform(_ action: () async throws -> Void) async {
        guard !changing, !loading else { return }
        changing = true
        defer { changing = false }
        do {
            try await action()
        } catch {
            self.error = error.localizedDescription
            // A season update can succeed only partially. Reconcile confirmed changes.
            try? await reloadTracking()
            return
        }
        do {
            try await reloadTracking()
            error = nil
        } catch {
            self.error = "Изменение сохранено, но обновить прогресс не удалось. Потяни страницу вниз для обновления."
        }
    }

    private func toggle(_ episode: Episode) async {
        guard episode.hasAired() else { return }
        let newValue = !watched.contains(episode.id)
        await perform {
            try await session.setWatched(showID: showID, episodeID: episode.id, watched: newValue)
            if newValue { watched.insert(episode.id) } else { watched.remove(episode.id) }
        }
    }

    private func markSeason(_ season: Season, watched newValue: Bool) async {
        await perform {
            defer { session.collectionDidChange() }
            let targets = season.episodes.filter { newValue ? ($0.hasAired() && !watched.contains($0.id)) : watched.contains($0.id) }
            for episode in targets {
                try await session.setWatched(showID: showID, episodeID: episode.id, watched: newValue, notifyCollection: false)
                if newValue { watched.insert(episode.id) } else { watched.remove(episode.id) }
            }
        }
    }
}
