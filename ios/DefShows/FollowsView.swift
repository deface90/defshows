import SwiftUI

/// FollowsView shows the current user's followers, who they follow, and incoming
/// follow requests (with approve/reject).
struct FollowsView: View {
    @EnvironmentObject private var session: Session
    enum Tab: String, CaseIterable { case followers, following, requests
        var title: String {
            switch self {
            case .followers: return "Подписчики"
            case .following: return "Подписки"
            case .requests: return "Запросы"
            }
        }
    }
    @State private var tab: Tab = .followers

    var body: some View {
        VStack(spacing: 0) {
            Picker("", selection: $tab) {
                ForEach(Tab.allCases, id: \.self) { Text($0.title).tag($0) }
            }
            .pickerStyle(.segmented)
            .padding()
            switch tab {
            case .followers: EdgeList(kind: .followers)
            case .following: EdgeList(kind: .following)
            case .requests: RequestsList()
            }
        }
        .navigationTitle("Подписки")
        .navigationBarTitleDisplayMode(.inline)
    }
}

private struct EdgeList: View {
    enum Kind { case followers, following }
    let kind: Kind
    @EnvironmentObject private var session: Session
    @State private var users: [FollowUser] = []
    @State private var loaded = false
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        List {
            if let error { Text(error).foregroundStyle(.red) }
            if loaded && users.isEmpty {
                ContentUnavailableView(kind == .followers ? "Нет подписчиков" : "Нет подписок", systemImage: "person.2")
            }
            ForEach(users) { u in
                NavigationLink {
                    PublicProfileView(user: PublicUser(id: u.id, displayName: u.displayName, isPublic: u.isPublic, showsCount: 0, isFollowing: nil, followersCount: nil, followingCount: nil))
                } label: {
                    HStack {
                        Image(systemName: u.isPublic ? "person.crop.circle" : "lock.circle").foregroundStyle(.secondary)
                        Text(u.displayName)
                    }
                }
                .swipeActions(edge: .trailing) {
                    if kind == .followers {
                        Button(role: .destructive) { Task { await removeFollower(u) } } label: {
                            Label("Убрать", systemImage: "person.badge.minus")
                        }.disabled(busy)
                    }
                }
            }
        }
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        guard let myID = session.currentUserID else { return }
        do {
            let result = kind == .followers ? try await session.followers(userID: myID) : try await session.following(userID: myID)
            users = result.users
            loaded = true
            error = nil
        } catch { self.error = error.localizedDescription }
    }

    /// Ejects an accepted follower (followers tab only); the list refetches so the
    /// removed user disappears.
    private func removeFollower(_ user: FollowUser) async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            try await session.removeFollower(userID: user.id)
            await load()
        } catch { self.error = error.localizedDescription }
    }
}

private struct RequestsList: View {
    @EnvironmentObject private var session: Session
    @State private var users: [FollowUser] = []
    @State private var loaded = false
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        List {
            if let error { Text(error).foregroundStyle(.red) }
            if loaded && users.isEmpty {
                ContentUnavailableView("Нет новых запросов", systemImage: "person.badge.clock")
            }
            ForEach(users) { u in
                HStack {
                    Text(u.displayName)
                    Spacer()
                    Button("Принять") { Task { await act { try await session.approveFollower(userID: u.id) } } }
                        .buttonStyle(.borderedProminent).controlSize(.small).disabled(busy)
                    Button("Отклонить") { Task { await act { try await session.rejectFollower(userID: u.id) } } }
                        .buttonStyle(.bordered).controlSize(.small).disabled(busy)
                }
            }
        }
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        do {
            users = try await session.incomingRequests().users
            loaded = true
            error = nil
        } catch { self.error = error.localizedDescription }
    }

    private func act(_ action: () async throws -> Void) async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do { try await action(); await load() }
        catch { self.error = error.localizedDescription }
    }
}
