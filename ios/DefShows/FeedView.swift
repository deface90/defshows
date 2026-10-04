import SwiftUI

/// FeedView is the home feed: the aggregated activity of everyone you follow.
struct FeedView: View {
    var body: some View {
        FeedList(source: .home)
            .navigationTitle("Лента")
            .navigationBarTitleDisplayMode(.inline)
    }
}

/// FeedList renders a cursor-paginated activity feed, reused by the home feed and a
/// profile's activity section.
struct FeedList: View {
    enum Source: Equatable { case home, profile(userID: Int) }
    let source: Source

    @EnvironmentObject private var session: Session
    @State private var cards: [FeedCard] = []
    @State private var cursor: String?
    @State private var loaded = false
    @State private var busy = false
    @State private var error: String?
    @State private var reportingActor: FollowUser?

    var body: some View {
        List {
            if let error { Text(error).foregroundStyle(.red) }
            if loaded && cards.isEmpty {
                ContentUnavailableView("Пока пусто", systemImage: "square.stack.3d.up", description: Text(emptyText))
            }
            ForEach(cards) { card in
                FeedCardRow(card: card)
                    .modifier(FeedActorActions(actor: card.actor,
                                               onBlock: { actor in Task { await block(actor) } },
                                               onReport: { actor in reportingActor = actor }))
            }
            if cursor != nil {
                Button("Показать ещё") { Task { await loadMore() } }.disabled(busy)
            }
        }
        .sheet(item: $reportingActor) { actor in
            ReportSheet(userID: actor.id, displayName: actor.displayName)
        }
        .task { await reload() }
        .refreshable { await reload() }
    }

    /// Blocks a feed-card actor; the server tears down the follow edge, so a reload
    /// drops their activity from the home feed. The busy flag is cleared before the
    /// reload so reload()'s `guard !busy` does not skip the refresh.
    private func block(_ actor: FollowUser) async {
        guard !busy else { return }
        busy = true
        do {
            try await session.block(userID: actor.id)
        } catch {
            self.error = error.localizedDescription
            busy = false
            return
        }
        busy = false
        await reload()
    }

    private var emptyText: String {
        switch source {
        case .home: return "Подпишитесь на кого-нибудь, чтобы видеть их активность."
        case .profile: return "У пользователя пока нет активности."
        }
    }

    private func page(cursor: String?) async throws -> FeedCardPage {
        switch source {
        case .home: return try await session.homeFeed(cursor: cursor)
        case .profile(let userID): return try await session.profileFeed(userID: userID, cursor: cursor)
        }
    }

    private func reload() async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            let p = try await page(cursor: nil)
            cards = p.cards
            cursor = p.nextCursor
            loaded = true
            error = nil
        } catch { self.error = error.localizedDescription }
    }

    private func loadMore() async {
        guard !busy, let c = cursor else { return }
        busy = true
        defer { busy = false }
        do {
            let p = try await page(cursor: c)
            cards.append(contentsOf: p.cards)
            cursor = p.nextCursor
            error = nil
        } catch { self.error = error.localizedDescription }
    }
}

/// FeedCardRow renders a single activity card; the actor line appears only on the home feed.
struct FeedCardRow: View {
    let card: FeedCard
    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            if let actor = card.actor {
                Text(actor.displayName).font(.caption).foregroundStyle(.secondary)
            }
            Text(card.show.title).font(.headline)
            Text(card.label).font(.subheadline).foregroundStyle(.orange)
        }
        .padding(.vertical, 4)
    }
}

/// Adds a long-press context menu (Block / Report) to a feed card when it carries
/// an actor (home feed only). Profile feeds have no actor, so the menu is absent.
private struct FeedActorActions: ViewModifier {
    let actor: FollowUser?
    let onBlock: (FollowUser) -> Void
    let onReport: (FollowUser) -> Void

    func body(content: Content) -> some View {
        if let actor {
            content.contextMenu {
                Button(role: .destructive) { onBlock(actor) } label: {
                    Label("Заблокировать \(actor.displayName)", systemImage: "hand.raised")
                }
                Button { onReport(actor) } label: {
                    Label("Пожаловаться", systemImage: "exclamationmark.bubble")
                }
            }
        } else {
            content
        }
    }
}
