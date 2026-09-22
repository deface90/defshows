import SwiftUI

struct UsersView: View {
    @EnvironmentObject private var session: Session
    @State private var query = ""
    @State private var appliedQuery = ""
    @State private var users: [PublicUser] = []
    @State private var page = 0
    @State private var total = 0
    @State private var busy = false
    @State private var loaded = false
    @State private var error: String?

    var body: some View {
        List {
            Section {
                HStack {
                    TextField("Имя пользователя", text: $query).submitLabel(.search)
                        .onSubmit { Task { await load(reset: true) } }
                    Button { Task { await load(reset: true) } } label: { Image(systemName: "magnifyingglass") }
                        .accessibilityLabel("Найти пользователей")
                }
            }.disabled(busy)
            if let error {
                Text(error).foregroundStyle(.red)
                Button("Повторить") { Task { await load(reset: true) } }.disabled(busy)
            }
            if busy { ProgressView() }
            if loaded && users.isEmpty {
                ContentUnavailableView("Никого не нашли", systemImage: "person.2")
            }
            ForEach(users) { user in
                NavigationLink { PublicProfileView(user: user) } label: {
                    HStack {
                        Image(systemName: user.isPublic ? "person.crop.circle" : "lock.circle")
                            .font(.title).foregroundStyle(.secondary)
                        VStack(alignment: .leading, spacing: 4) {
                            Text(user.displayName).font(.headline)
                            Text(user.isPublic ? "Сериалов: \(user.showsCount)" : "Закрытый профиль")
                                .font(.caption).foregroundStyle(.secondary)
                        }
                    }
                }
            }
            if users.count < total {
                Button("Показать ещё") { Task { await load(reset: false) } }.disabled(busy)
            }
        }
        .navigationTitle("Пользователи")
        .task { if !loaded { await load(reset: true) } }
        .refreshable { await load(reset: true) }
    }

    private func load(reset: Bool) async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        let nextPage = reset ? 1 : page + 1
        let term = reset ? String(query.trimmingCharacters(in: .whitespacesAndNewlines).prefix(100)) : appliedQuery
        do {
            let result: UserDirectory = try await session.get("users", query: [
                URLQueryItem(name: "q", value: term), URLQueryItem(name: "page", value: String(nextPage)),
                URLQueryItem(name: "page_size", value: "20")
            ])
            let existing = reset ? Set<Int>() : Set(users.map(\.id))
            users = (reset ? [] : users) + result.users.filter { !existing.contains($0.id) }
            page = result.page
            total = result.total
            appliedQuery = term
            loaded = true
            error = nil
        } catch { self.error = error.localizedDescription }
    }
}

struct PublicProfileView: View {
    let user: PublicUser
    @EnvironmentObject private var session: Session
    @State private var shows: [PublicTrackedShow] = []
    @State private var profile: PublicUser?
    @State private var busy = false
    @State private var loaded = false
    @State private var error: String?

    var body: some View {
        List {
            if let error {
                Text(error).foregroundStyle(.red)
                Button("Повторить") { Task { await load() } }.disabled(busy)
            }
            if busy { ProgressView() }
            if let profile, !profile.isPublic {
                ContentUnavailableView("Закрытый профиль", systemImage: "lock", description: Text("Пользователь скрыл свою коллекцию."))
            } else if loaded && shows.isEmpty {
                ContentUnavailableView("Пока нет сериалов", systemImage: "tv")
            }
            ForEach(shows) { item in
                NavigationLink { CatalogShowView(tmdbID: item.show.tmdbId) } label: {
                    CatalogRow(title: item.show.title, poster: item.show.posterUrl,
                               subtitle: "\(item.userShow.statusTitle) · \(item.progress.watched) из \(item.progress.total) серий")
                }
            }
        }
        .navigationTitle(profile?.displayName ?? user.displayName)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            let profile: PublicUser = try await session.get("users/\(user.id)")
            self.profile = profile
            shows = []
            if profile.isPublic {
                let result: PublicCollection = try await session.get("users/\(user.id)/shows")
                shows = result.tracked
            }
            loaded = true
            error = nil
        } catch { self.error = error.localizedDescription }
    }
}
