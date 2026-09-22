import Foundation

struct CatalogResults: Decodable {
    let results: [CatalogItem]
    let page: Int?
    let totalPages: Int?
}
struct CatalogItem: Decodable, Identifiable {
    let tmdbId: Int
    let title: String
    let overview: String?
    let posterUrl: String?
    let voteAverage: Double?
    var id: Int { tmdbId }
}
struct CatalogDetail: Decodable {
    let id: Int
    let title: String
    let overview: String?
    let posterUrl: String?
    let seasons: [Season]?
}
struct DiscoveryFilters: Decodable {
    let genres: [CatalogGenre]
    let countries: [CatalogCountry]
}
struct CatalogGenre: Decodable, Identifiable {
    let id: Int
    let name: String
}
struct CatalogCountry: Decodable, Identifiable {
    let code: String
    let name: String
    var id: String { code }
}
struct NotificationFeed: Decodable { let notifications: [FeedItem] }
struct FeedItem: Decodable, Identifiable {
    let id: Int
    let type: String
    var read: Bool
    let payload: String
    let createdAt: String
    var title: String {
        switch type {
        case "episode_released": return "Новая серия"
        case "episode_upcoming": return "Скоро серия"
        case "season_upcoming": return "Скоро сезон"
        default: return "Уведомление"
        }
    }
    var dateText: String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let date = formatter.date(from: createdAt) ?? ISO8601DateFormatter().date(from: createdAt)
        return date?.formatted(date: .abbreviated, time: .shortened) ?? createdAt
    }
}
struct UserDirectory: Decodable {
    let users: [PublicUser]
    let total: Int
    let page: Int
    let pageSize: Int
}
struct PublicUser: Decodable, Identifiable {
    let id: Int
    let displayName: String
    let isPublic: Bool
    let showsCount: Int
}
// Public profiles deliberately omit watched episode IDs.
struct PublicCollection: Decodable { let tracked: [PublicTrackedShow] }
struct PublicTrackedShow: Decodable, Identifiable {
    let show: ShowReference
    let userShow: UserShow
    let progress: PublicProgress
    var id: Int { show.id }
}
struct PublicProgress: Decodable { let watched: Int; let total: Int }
struct AccountSettings: Decodable {
    var timezone: String
    var isPublic: Bool
}
struct NotificationPreferences: Decodable {
    var episodeRelease: Bool
    var seasonStart: Bool
    var weeklyDigest: Bool
    var leadTimeHours: Int
}
