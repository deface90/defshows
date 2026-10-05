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
    // Optimistic per-user follow state, overriding the directory's is_following until the
    // next reset reload. followBusy disables a row's button while its request is in flight.
    @State private var followState: [Int: String] = [:]
    @State private var followBusy: Set<Int> = []

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
                        if session.currentUserID != user.id {
                            Spacer(minLength: 8)
                            followButton(for: user)
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

    /// A borderless follow/unfollow button for a directory row. Borderless so it stays
    /// independently tappable inside the row's NavigationLink (a tap toggles the follow,
    /// it does not open the profile). State comes from the optimistic override, else the
    /// directory's is_following, else "none".
    @ViewBuilder
    private func followButton(for user: PublicUser) -> some View {
        let state = followState[user.id] ?? user.isFollowing ?? "none"
        Button {
            Task { await toggleFollow(user, from: state) }
        } label: {
            switch state {
            case "accepted": Text("Вы подписаны")
            case "pending": Text("Запрос отправлен")
            default: Text("Подписаться")
            }
        }
        .buttonStyle(.borderless)
        .font(.caption)
        .disabled(followBusy.contains(user.id))
    }

    /// Follows (from "none") or cancels/unfollows (from "pending"/"accepted"), then stores
    /// the resulting state optimistically so the row updates without a reload.
    private func toggleFollow(_ user: PublicUser, from state: String) async {
        guard !followBusy.contains(user.id) else { return }
        followBusy.insert(user.id)
        defer { followBusy.remove(user.id) }
        do {
            if state == "none" {
                let result = try await session.follow(userID: user.id)
                followState[user.id] = result.status
            } else {
                try await session.unfollow(userID: user.id)
                followState[user.id] = "none"
            }
            error = nil
        } catch { self.error = error.localizedDescription }
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
            if reset { followState.removeAll() }
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
    // The profile endpoint returns 404 for a blocked pair (either direction), a deleted
    // account, or an unknown id — it never confirms existence. `unavailable` captures that
    // neutral "can't show this profile" state. `outgoingBlock` is tracked separately,
    // derived from the viewer's own blocked-list, so Unblock is offered only when the
    // viewer actually has an outgoing block (a 404 alone must not assert one).
    @State private var unavailable = false
    @State private var outgoingBlock = false
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
            if unavailable {
                if outgoingBlock {
                    ContentUnavailableView("Пользователь заблокирован", systemImage: "hand.raised",
                                           description: Text("Вы не видите коллекцию и активность заблокированного пользователя."))
                } else {
                    ContentUnavailableView("Профиль недоступен", systemImage: "eye.slash",
                                           description: Text("Этот профиль сейчас недоступен."))
                }
            }
            if let profile, !unavailable {
                Section {
                    if let followers = profile.followersCount, let following = profile.followingCount {
                        Text("\(followers) подписчиков · \(following) подписок")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                    if !isSelf { followButton(profile) }
                }
            }
            if let profile, canView, !unavailable {
                Section {
                    NavigationLink {
                        FeedList(source: .profile(userID: user.id))
                            .navigationTitle("Активность")
                            .navigationBarTitleDisplayMode(.inline)
                    } label: { Label("Активность", systemImage: "square.stack.3d.up") }
                }
            }
            if !unavailable {
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
                    ModerationMenu(blocked: outgoingBlock,
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
        do {
            if block {
                try await session.block(userID: user.id)
                outgoingBlock = true
                unavailable = true
                profile = nil
                shows = []
                error = nil
                busy = false
            } else {
                try await session.unblock(userID: user.id)
                outgoingBlock = false
                error = nil
                // Clear busy before reload so load()'s `guard !busy` does not skip it.
                busy = false
                await load()
            }
        } catch {
            self.error = error.localizedDescription
            busy = false
        }
    }

    /// hasOutgoingBlock reports whether the viewer has blocked this user, derived from
    /// the viewer's own blocked-list. Used to disambiguate a profile 404 (which could be a
    /// block in *either* direction, a deleted account, or an unknown id) so Unblock is
    /// offered only for an actual outgoing block. Best-effort: on error it reports false.
    private func hasOutgoingBlock() async -> Bool {
        guard session.currentUserID != nil else { return false }
        // Page through the blocked-list until the user is found or the list is exhausted —
        // checking only the first page would misclassify an older outgoing block as "no
        // outgoing block" and wrongly offer Block (not Unblock). Bounded by the real total.
        let pageSize = 50
        var page = 1
        var seen = 0
        while true {
            guard let list = try? await session.blocks(page: page, pageSize: pageSize) else { return false }
            if list.users.contains(where: { $0.id == user.id }) { return true }
            seen += list.users.count
            if list.users.isEmpty || seen >= list.total { return false }
            page += 1
        }
    }

    private func load() async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            let fetched: PublicUser = try await session.get("users/\(user.id)")
            self.profile = fetched
            unavailable = false
            outgoingBlock = false
            shows = []
            if fetched.isPublic || fetched.isFollowing == "accepted" || isSelf {
                let result: PublicCollection = try await session.get("users/\(user.id)/shows")
                shows = result.tracked
            }
            loaded = true
            error = nil
        } catch let failure as HTTPFailure where failure.status == 404 {
            // The server returns 404 for a blocked pair (either direction), a deleted
            // account, or an unknown id — it never confirms existence. Surface a neutral
            // "unavailable" state; only assert an outgoing block (→ offer Unblock) when the
            // viewer's own blocked-list actually contains this user.
            unavailable = true
            outgoingBlock = await hasOutgoingBlock()
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
