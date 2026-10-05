import SwiftUI

struct LibraryFilterOption: Identifiable {
    let value: String
    let label: String
    var id: String { value }
}

enum LibrarySort: String, CaseIterable, Identifiable {
    case progress, title, dateAdded
    var id: String { rawValue }
    var label: String {
        switch self {
        case .progress: return "По прогрессу"
        case .title: return "По названию"
        case .dateAdded: return "По дате добавления"
        }
    }
}

struct LibraryView: View {
    @EnvironmentObject private var session: Session
    @State private var shows: [TrackedShow] = []
    @State private var query = ""
    @State private var statusFilter = "watching"
    @State private var airingFilter = ""
    @State private var sort: LibrarySort = .progress
    @State private var reversed = false
    @State private var showingStats = false
    @State private var showingNotifications = false
    @State private var loading = false
    @State private var loaded = false
    @State private var error: String?

    // The status filter uses "" to mean "all"; the rest map to UserShow.status.
    private let statusOptions: [LibraryFilterOption] = [
        .init(value: "", label: "Все"), .init(value: "watching", label: "Смотрю"),
        .init(value: "plan_to_watch", label: "В планах"), .init(value: "on_hold", label: "На паузе"),
        .init(value: "completed", label: "Просмотрено"), .init(value: "dropped", label: "Брошено"),
    ]
    private let airingOptions: [LibraryFilterOption] = [
        .init(value: "", label: "Любой статус выхода"), .init(value: "airing", label: "Идёт"),
        .init(value: "between_seasons", label: "Между сезонами"), .init(value: "not_started", label: "Не начат"),
        .init(value: "ended", label: "Завершён"),
    ]

    private var counts: [String: Int] {
        var map: [String: Int] = ["": shows.count]
        for option in statusOptions where !option.value.isEmpty {
            map[option.value] = shows.filter { $0.userShow.status == option.value }.count
        }
        return map
    }

    private func pct(_ item: TrackedShow) -> Double {
        item.progress.total > 0 ? Double(item.progress.watched) / Double(item.progress.total) : 0
    }

    private var visible: [TrackedShow] {
        var list = shows
        if !statusFilter.isEmpty { list = list.filter { $0.userShow.status == statusFilter } }
        if !airingFilter.isEmpty { list = list.filter { $0.show.airingStatus == airingFilter } }
        let needle = query.trimmingCharacters(in: .whitespacesAndNewlines)
        if !needle.isEmpty {
            list = list.filter {
                $0.show.title.localizedCaseInsensitiveContains(needle)
                    || ($0.show.originalTitle?.localizedCaseInsensitiveContains(needle) ?? false)
            }
        }
        switch sort {
        case .title:
            list.sort { $0.show.title.localizedCaseInsensitiveCompare($1.show.title) == .orderedAscending }
        case .progress:
            list.sort { pct($0) < pct($1) }
        case .dateAdded:
            break // server order: recently added first
        }
        if reversed { list.reverse() }
        return list
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
                } else if loaded && visible.isEmpty {
                    if query.isEmpty {
                        ContentUnavailableView("Ничего не найдено", systemImage: "line.3.horizontal.decrease.circle", description: Text("Измени фильтры."))
                    } else {
                        ContentUnavailableView.search(text: query)
                    }
                }
                ForEach(visible) { item in
                    NavigationLink {
                        ShowView(showID: item.id)
                    } label: {
                        LibraryRow(item: item, posterURL: session.posterURL(item.show.posterUrl))
                    }
                }
            }
            .navigationTitle("Мои сериалы")
            .searchable(text: $query, prompt: "Найти в коллекции")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button { showingStats = true } label: {
                        Image(systemName: "chart.bar")
                    }
                    .accessibilityLabel("Статистика")
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button { showingNotifications = true } label: {
                        Image(systemName: session.unreadNotifications > 0 ? "bell.badge" : "bell")
                    }
                    .accessibilityLabel(session.unreadNotifications > 0 ? "Уведомления, есть новые" : "Уведомления")
                }
                ToolbarItem(placement: .topBarTrailing) {
                    filterMenu
                }
            }
            .sheet(isPresented: $showingStats) { StatsSheet() }
            .sheet(isPresented: $showingNotifications, onDismiss: { Task { await session.refreshUnreadCount() } }) {
                NavigationStack {
                    NotificationsView()
                        .toolbar {
                            ToolbarItem(placement: .topBarLeading) {
                                Button("Готово") { showingNotifications = false }
                            }
                        }
                }
            }
            .refreshable { await load() }
            .onAppear {
                Task { await load() }
                Task { await session.refreshUnreadCount() }
            }
            .onChange(of: session.collectionRevision) { _, _ in Task { await load() } }
        }
    }

    private var filterMenu: some View {
        Menu {
            Picker("Статус", selection: $statusFilter) {
                ForEach(statusOptions) { option in
                    Text("\(option.label) (\(counts[option.value] ?? 0))").tag(option.value)
                }
            }
            Picker("Статус выхода", selection: $airingFilter) {
                ForEach(airingOptions) { option in
                    Text(option.label).tag(option.value)
                }
            }
            Picker("Сортировка", selection: $sort) {
                ForEach(LibrarySort.allCases) { option in
                    Text(option.label).tag(option)
                }
            }
            Toggle("Обратный порядок", isOn: $reversed)
        } label: {
            Image(systemName: "line.3.horizontal.decrease.circle")
        }
        .accessibilityLabel("Фильтры и сортировка")
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

/// A single collection row: poster, titles, badges and progress.
private struct LibraryRow: View {
    let item: TrackedShow
    let posterURL: URL?

    var body: some View {
        HStack(spacing: 14) {
            AsyncImage(url: posterURL) { image in
                image.resizable().scaledToFill()
            } placeholder: {
                Rectangle().fill(.quaternary).overlay { Image(systemName: "tv").foregroundStyle(.secondary) }
            }
            .frame(width: 62, height: 92).clipShape(RoundedRectangle(cornerRadius: 8))
            VStack(alignment: .leading, spacing: 7) {
                Text(item.show.title).font(.headline)
                if let original = item.show.distinctOriginalTitle {
                    Text(original).font(.caption).foregroundStyle(.secondary)
                }
                Text(item.userShow.statusTitle).font(.caption).foregroundStyle(.secondary)
                badges
                ProgressView(value: Double(min(item.progress.watched, item.progress.total)), total: Double(max(item.progress.total, 1)))
                Text("\(item.progress.watched) из \(item.progress.total) вышедших серий")
                    .font(.caption).foregroundStyle(.secondary)
            }
            if item.userShow.favorite { Image(systemName: "heart.fill").foregroundStyle(.pink).accessibilityLabel("Избранное") }
        }.padding(.vertical, 4)
    }

    @ViewBuilder private var badges: some View {
        let unwatched = item.progress.unwatched ?? 0
        if item.progress.newFullSeason != nil || unwatched > 0 || item.show.airingStatusTitle != nil {
            HStack(spacing: 6) {
                if let airing = item.show.airingStatusTitle {
                    badge(airing, color: .secondary)
                }
                if let season = item.progress.newFullSeason {
                    badge("🆕 Новый сезон \(season)", color: .green)
                }
                if unwatched > 0 {
                    badge("\(unwatched) к просмотру", color: .orange)
                }
            }
        }
    }

    private func badge(_ text: String, color: Color) -> some View {
        Text(text)
            .font(.caption2).fontWeight(.semibold)
            .padding(.horizontal, 7).padding(.vertical, 3)
            .background(color.opacity(0.18), in: Capsule())
            .foregroundStyle(color)
    }
}

/// StatsSheet shows aggregate viewing statistics from GET /me/stats.
private struct StatsSheet: View {
    @EnvironmentObject private var session: Session
    @Environment(\.dismiss) private var dismiss
    @State private var stats: Stats?
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                if let error {
                    Text(error).foregroundStyle(.red)
                    Button("Повторить") { Task { await load() } }
                } else if let stats {
                    row("Добавлено сериалов", "\(stats.showsTracked)")
                    row("Просмотрено сериалов", "\(stats.showsCompleted)")
                    row("Просмотрено сезонов", "\(stats.seasonsWatched)")
                    row("Просмотрено эпизодов", "\(stats.episodesWatched)")
                    row("Потрачено времени", stats.watchTimeText)
                } else {
                    ProgressView()
                }
            }
            .navigationTitle("Статистика")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Готово") { dismiss() }
                }
            }
            .task { await load() }
        }
    }

    private func row(_ label: String, _ value: String) -> some View {
        HStack {
            Text(label)
            Spacer()
            Text(value).foregroundStyle(.secondary).monospacedDigit()
        }
    }

    private func load() async {
        do {
            stats = try await session.get("me/stats")
            error = nil
        } catch { self.error = error.localizedDescription }
    }
}
