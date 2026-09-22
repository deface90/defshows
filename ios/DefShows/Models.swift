import Foundation

struct Tokens: Codable {
    let accessToken: String
    let refreshToken: String
}
struct AuthResponse: Decodable { let tokens: Tokens }
struct TrackedList: Decodable { let tracked: [TrackedShow] }
struct TrackedShow: Decodable, Identifiable {
    let show: ShowReference
    let userShow: UserShow
    let progress: WatchProgress
    var id: Int { show.id }
}
struct ShowReference: Decodable {
    let tmdbId: Int
    let id: Int
    let title: String
    let posterUrl: String?
}
struct UserShow: Decodable {
    let status: String
    let favorite: Bool
    var statusTitle: String {
        switch status {
        case "watching": return "Смотрю"
        case "plan_to_watch": return "В планах"
        case "on_hold": return "На паузе"
        case "completed": return "Просмотрено"
        case "dropped": return "Брошено"
        default: return status
        }
    }
}
struct WatchProgress: Decodable {
    let watched: Int
    let total: Int
    let watchedEpisodeIds: [Int]
}
struct ShowDetail: Decodable {
    let title: String
    let overview: String?
    let seasons: [Season]?
}
struct Season: Decodable, Identifiable {
    let id: Int
    let seasonNumber: Int
    let name: String
    let episodes: [Episode]
}
struct Episode: Decodable, Identifiable {
    let id: Int
    let episodeNumber: Int
    let name: String
    let airDate: String?
}
struct APIError: LocalizedError {
    let message: String
    var errorDescription: String? { message }
}

// Keep the same order in tracked and catalog detail screens.
extension Array where Element == Season {
    var newestFirst: [Season] { sorted { $0.seasonNumber > $1.seasonNumber } }
}
