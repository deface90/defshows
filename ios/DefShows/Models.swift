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
    let nextUnwatchedEpisodeId: Int?
}
struct ShowDetail: Decodable {
    let id: Int
    let title: String
    let originalTitle: String?
    let posterUrl: String?
    let firstAirDate: String?
    let nextEpisodeAirDate: String?
    let airingStatus: String?
    let voteAverage: Double?
    let voteCount: Int?
    let genres: [CatalogGenre]?
    let ratings: [ShowRating]?
    let imdbUrl: String?
    let wikipediaUrl: String?
    let overview: String?
    let seasons: [Season]?
}
struct Season: Decodable, Identifiable {
    let id: Int
    let seasonNumber: Int
    let name: String
    let episodes: [Episode]
    let voteAverage: Double?
}
struct Episode: Decodable, Identifiable {
    let id: Int
    let seasonNumber: Int
    let episodeNumber: Int
    let name: String
    let airDate: String?
    let runtime: Int?
    let voteAverage: Double?
    let voteCount: Int?

    var code: String { String(format: "S%02dE%02d", seasonNumber, episodeNumber) }
    func hasAired(on today: Date = Date()) -> Bool {
        guard let airDate else { return false }
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.isLenient = false
        guard let date = formatter.date(from: airDate) else { return false }
        return date <= Calendar.current.startOfDay(for: today)
    }
}
struct APIError: LocalizedError {
    let message: String
    var errorDescription: String? { message }
}

// Keep the same order in tracked and catalog detail screens.
extension Array where Element == Season {
    var newestFirst: [Season] { sorted { $0.seasonNumber > $1.seasonNumber } }
}

struct ShowRating: Decodable {
    let source: String
    let value: String
    let votes: Int?
}

struct NoteList: Decodable { let notes: [ShowNote] }
struct ShowNote: Decodable, Identifiable {
    let id: Int
    let scope: String
    let seasonNumber: Int?
    let episodeNumber: Int?
    let body: String
    var scopeTitle: String {
        switch scope {
        case "season": return "Сезон \(seasonNumber ?? 0)"
        case "episode": return "S\(seasonNumber ?? 0)E\(episodeNumber ?? 0)"
        default: return "Сериал"
        }
    }
}
