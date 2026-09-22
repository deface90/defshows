import SwiftUI

struct ShowView: View {
    let showID: Int
    @EnvironmentObject private var session: Session
    @State private var detail: ShowDetail?
    @State private var watched: Set<Int> = []
    @State private var loading = false
    @State private var changing = false
    @State private var error: String?

    var body: some View {
        List {
            if let error {
                Section {
                    Text(error).foregroundStyle(.red)
                    Button("Обновить") { Task { await load() } }.disabled(changing || loading)
                }
            }
            if let detail {
                if let overview = detail.overview, !overview.isEmpty {
                    Section("О сериале") { Text(overview).foregroundStyle(.secondary) }
                }
                if (detail.seasons ?? []).isEmpty {
                    ContentUnavailableView("Серии пока не загружены", systemImage: "tv")
                }
                ForEach((detail.seasons ?? []).newestFirst) { season in
                    Section(season.name.isEmpty ? "Сезон \(season.seasonNumber)" : season.name) {
                        ForEach(season.episodes.sorted { $0.episodeNumber < $1.episodeNumber }) { episode in
                            Button {
                                Task { await toggle(episode) }
                            } label: {
                                HStack(spacing: 12) {
                                    Image(systemName: watched.contains(episode.id) ? "checkmark.circle.fill" : "circle")
                                        .font(.title2).foregroundStyle(watched.contains(episode.id) ? Color.indigo : Color.secondary)
                                    VStack(alignment: .leading, spacing: 4) {
                                        Text("\(episode.episodeNumber). \(episode.name)").foregroundStyle(.primary)
                                        if let date = episode.airDate {
                                            Text(date).font(.caption).foregroundStyle(.secondary)
                                        }
                                    }
                                    Spacer()
                                }.padding(.vertical, 3).contentShape(Rectangle())
                            }
                            .buttonStyle(.plain)
                            .disabled(changing || loading)
                            .accessibilityLabel("\(episode.name), \(watched.contains(episode.id) ? "просмотрено" : "не просмотрено")")
                            .accessibilityHint("Изменить отметку просмотра")
                        }
                    }
                }
            } else if loading { ProgressView("Загружаем серии…") }
        }
        .navigationTitle(detail?.title ?? "Сериал")
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { await load() }
        .toolbar { if changing { ProgressView() } }
    }

    private func load() async {
        guard !loading, !changing else { return }
        loading = true
        defer { loading = false }
        do {
            let show: ShowDetail = try await session.get("shows/\(showID)")
            let tracked: TrackedShow = try await session.get("me/shows/\(showID)")
            detail = show
            watched = Set(tracked.progress.watchedEpisodeIds)
            error = nil
        } catch is CancellationError { }
        catch { self.error = error.localizedDescription }
    }

    private func toggle(_ episode: Episode) async {
        guard !changing, !loading else { return }
        changing = true
        defer { changing = false }
        let newValue = !watched.contains(episode.id)
        do {
            try await session.setWatched(showID: showID, episodeID: episode.id, watched: newValue)
            if newValue { watched.insert(episode.id) } else { watched.remove(episode.id) }
            error = nil
        } catch { self.error = error.localizedDescription }
    }
}
