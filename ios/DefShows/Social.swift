import Foundation

/// Result of a follow action: "pending" (private target) or "accepted" (public).
struct FollowResult: Decodable { let status: String }

/// A user as shown in follower/following/incoming lists.
struct FollowUser: Decodable, Identifiable {
    let id: Int
    let displayName: String
    let isPublic: Bool
}

struct FollowUserList: Decodable {
    let users: [FollowUser]
    let total: Int
}

/// A show reference inside a feed card.
struct FeedShowRef: Decodable {
    let id: Int
    let tmdbId: Int
    let title: String
    let posterUrl: String?
}

/// An episode reference inside a grouped watched_episode card.
struct FeedEpisodeRef: Decodable {
    let seasonNumber: Int
    let episodeNumber: Int
}

/// One activity feed card. `actor` is present only on the home feed.
struct FeedCard: Decodable, Identifiable {
    let actor: FollowUser?
    let type: String
    let show: FeedShowRef
    let seasonNumber: Int?
    let episodes: [FeedEpisodeRef]?
    let count: Int
    let rating: Int?
    let createdAt: String

    var id: String { "\(type)-\(createdAt)-\(show.id)" }

    private func code(_ season: Int, _ episode: Int) -> String {
        String(format: "S%02dE%02d", season, episode)
    }

    /// Short Russian action phrase; grouped watched episodes collapse to a range.
    var label: String {
        switch type {
        case "watched_episode":
            let eps = (episodes ?? []).sorted {
                $0.seasonNumber != $1.seasonNumber
                    ? $0.seasonNumber < $1.seasonNumber
                    : $0.episodeNumber < $1.episodeNumber
            }
            guard let first = eps.first else { return "Просмотр серий" }
            if eps.count == 1 { return "Серия \(code(first.seasonNumber, first.episodeNumber))" }
            let last = eps[eps.count - 1]
            let end = first.seasonNumber == last.seasonNumber
                ? String(format: "E%02d", last.episodeNumber)
                : code(last.seasonNumber, last.episodeNumber)
            return "Серии \(code(first.seasonNumber, first.episodeNumber))–\(end) · \(count) серий"
        case "finished_season":
            return seasonNumber.map { "Сезон \($0) завершён" } ?? "Сезон завершён"
        case "finished_show":
            return "Сериал завершён"
        case "added_show":
            return "Добавлен в коллекцию"
        case "rated_show":
            return rating.map { "Оценка \($0)/10" } ?? "Оценка снята"
        default:
            return ""
        }
    }
}

/// One page of a cursor-paginated feed.
struct FeedCardPage: Decodable {
    let cards: [FeedCard]
    let nextCursor: String?
}

/// A reason a user can pick when reporting another user. Raw values match the
/// server `reason` enum (social.yaml `ReportRequest`).
enum ReportReason: String, CaseIterable, Identifiable {
    case spam
    case harassment
    case inappropriate
    case other

    var id: String { rawValue }

    /// Russian label shown in the report picker.
    var title: String {
        switch self {
        case .spam: return "Спам"
        case .harassment: return "Оскорбления или травля"
        case .inappropriate: return "Неприемлемый контент"
        case .other: return "Другое"
        }
    }
}

/// A report filed against another user (echoed back by POST /me/reports).
struct Report: Decodable, Identifiable {
    let id: Int
    let reporterId: Int
    let targetUserId: Int
    let reason: String
    let note: String
    let status: String
    let createdAt: String
}
