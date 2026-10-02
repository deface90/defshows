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
    @State private var blocked = false
    @State private var error: String?
    @State private var reporting = false

    private var isSelf: Bool { session.currentUserID == user.id }
    private var canView: Bool {
        guard let profile else { return false }
        return profile.isPublic || profile.isFollowing == "accepted" || isSelf
    }

    var body: some View {
        List {
            if let error {
                Text(error).foregroundStyle(.red)
                Button("Повторить") { Task { await load() } }.disabled(busy)
            }
            if busy { ProgressView() }
            if blocked {
                ContentUnavailableView("Пользователь заблокирован", systemImage: "hand.raised",
                                       description: Text("Вы не видите коллекцию и активность заблокированного пользователя."))
            }
            if let profile, !blocked {
                Section {
                    if let followers = profile.followersCount, let following = profile.followingCount {
                        Text("\(followers) подписчиков · \(following) подписок")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                    if !isSelf { followButton(profile) }
                }
            }
            if let profile, canView, !blocked {
                Section {
                    NavigationLink {
                        FeedList(source: .profile(userID: user.id))
                            .navigationTitle("Активность")
                            .navigationBarTitleDisplayMode(.inline)
                    } label: { Label("Активность", systemImage: "square.stack.3d.up") }
                }
            }
            if !blocked {
                if let profile, !canView {
                    ContentUnavailableView("Закрытый профиль", systemImage: "lock", description: Text("Коллекция видна только одобренным подписчикам."))
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
        }
        .navigationTitle(profile?.displayName ?? user.displayName)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            if !isSelf {
                ToolbarItem(placement: .topBarTrailing) {
                    ModerationMenu(blocked: blocked,
                                   onBlock: { Task { await setBlock(true) } },
                                   onUnblock: { Task { await setBlock(false) } },
                                   onReport: { reporting = true })
                        .disabled(busy)
                }
            }
        }
        .sheet(isPresented: $reporting) {
            ReportSheet(userID: user.id, displayName: profile?.displayName ?? user.displayName)
        }
        .task { await load() }
        .refreshable { await load() }
    }

    @ViewBuilder private func followButton(_ profile: PublicUser) -> some View {
        switch profile.isFollowing {
        case "accepted":
            Button("Вы подписаны") { Task { await toggleFollow(unfollow: true) } }.disabled(busy)
        case "pending":
            Button("Запрос отправлен") { Task { await toggleFollow(unfollow: true) } }.disabled(busy)
        default:
            Button("Подписаться") { Task { await toggleFollow(unfollow: false) } }.disabled(busy)
        }
    }

    private func toggleFollow(unfollow: Bool) async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            if unfollow { try await session.unfollow(userID: user.id) }
            else { _ = try await session.follow(userID: user.id) }
            await load()
        } catch { self.error = error.localizedDescription }
    }

    private func setBlock(_ block: Bool) async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            if block {
                try await session.block(userID: user.id)
                blocked = true
                profile = nil
                shows = []
            } else {
                try await session.unblock(userID: user.id)
                blocked = false
                await load()
            }
            error = nil
        } catch { self.error = error.localizedDescription }
    }

    private func load() async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            let fetched: PublicUser = try await session.get("users/\(user.id)")
            self.profile = fetched
            blocked = false
            shows = []
            if fetched.isPublic || fetched.isFollowing == "accepted" || isSelf {
                let result: PublicCollection = try await session.get("users/\(user.id)/shows")
                shows = result.tracked
            }
            loaded = true
            error = nil
        } catch let failure as HTTPFailure where failure.status == 404 {
            // The server returns 404 for a blocked profile (either direction) to avoid
            // confirming existence; surface it as the blocked state rather than an error.
            blocked = true
            profile = nil
            shows = []
            loaded = true
            error = nil
        } catch { self.error = error.localizedDescription }
    }
}

/// An overflow menu with Block/Unblock + Report actions, shared by the public
/// profile and feed-card actors. Block is confirmed with a dialog; Report opens
/// the caller-supplied sheet via `onReport`.
struct ModerationMenu: View {
    let blocked: Bool
    let onBlock: () -> Void
    let onUnblock: () -> Void
    let onReport: () -> Void
    @State private var confirmingBlock = false

    var body: some View {
        Menu {
            if blocked {
                Button { onUnblock() } label: { Label("Разблокировать", systemImage: "hand.raised.slash") }
            } else {
                Button(role: .destructive) { confirmingBlock = true } label: {
                    Label("Заблокировать", systemImage: "hand.raised")
                }
            }
            Button { onReport() } label: { Label("Пожаловаться", systemImage: "exclamationmark.bubble") }
        } label: {
            Image(systemName: "ellipsis.circle").accessibilityLabel("Действия модерации")
        }
        .confirmationDialog("Заблокировать пользователя?", isPresented: $confirmingBlock, titleVisibility: .visible) {
            Button("Заблокировать", role: .destructive) { onBlock() }
            Button("Отмена", role: .cancel) {}
        } message: {
            Text("Вы перестанете видеть друг друга: подписки снимаются, повторная подписка невозможна.")
        }
    }
}

/// A sheet for filing a moderation report: pick a reason, add an optional note,
/// and submit to POST /me/reports.
struct ReportSheet: View {
    let userID: Int
    let displayName: String
    @EnvironmentObject private var session: Session
    @Environment(\.dismiss) private var dismiss
    @State private var reason: ReportReason = .spam
    @State private var note = ""
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Picker("Причина", selection: $reason) {
                        ForEach(ReportReason.allCases) { Text($0.title).tag($0) }
                    }
                } header: {
                    Text("Жалоба на \(displayName)")
                }
                Section {
                    TextField("Комментарий (необязательно)", text: $note, axis: .vertical)
                        .lineLimit(3...6)
                } footer: {
                    Text("Жалоба отправляется модераторам. Пользователь об этом не узнает.")
                }
                if let error { Section { Text(error).foregroundStyle(.red) } }
            }
            .navigationTitle("Пожаловаться")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Отмена") { dismiss() }.disabled(busy)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Отправить") { Task { await submit() } }.disabled(busy)
                }
            }
            .overlay { if busy { ProgressView() } }
        }
    }

    private func submit() async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            try await session.report(userID: userID, reason: reason, note: note)
            dismiss()
        } catch { self.error = error.localizedDescription }
    }
}
