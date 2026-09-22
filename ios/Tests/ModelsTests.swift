import XCTest
@testable import DefShowsModels

final class ModelsTests: XCTestCase {
    private func decode<T: Decodable>(_ type: T.Type, _ json: String) throws -> T {
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return try decoder.decode(type, from: Data(json.utf8))
    }

    func testSearchResponseDoesNotRequirePaginationOrPosters() throws {
        let result = try decode(CatalogResults.self, #"{"results":[{"tmdb_id":42,"title":"Тест"}]}"#)
        XCTAssertEqual(result.results.first?.id, 42)
        XCTAssertNil(result.results.first?.posterUrl)
        XCTAssertNil(result.totalPages)
    }

    func testDiscoveryDecodesPaginationAndRating() throws {
        let result = try decode(CatalogResults.self, #"{"results":[{"tmdb_id":42,"title":"Тест","vote_average":8.5}],"page":2,"total_pages":10,"total_results":200}"#)
        XCTAssertEqual(result.page, 2)
        XCTAssertEqual(result.totalPages, 10)
        XCTAssertEqual(result.results.first?.voteAverage, 8.5)
    }

    func testPublicProgressDoesNotRequirePrivateEpisodeIDs() throws {
        let collection = try decode(PublicCollection.self, #"{"tracked":[{"show":{"id":1,"tmdb_id":42,"title":"Тест","airing_status":"ended"},"user_show":{"id":3,"show_id":1,"status":"watching","favorite":false},"progress":{"watched":2,"total":5}}]}"#)
        XCTAssertEqual(collection.tracked.first?.progress.watched, 2)
        XCTAssertEqual(collection.tracked.first?.show.tmdbId, 42)
    }

    func testSeasonsNewestFirstWithoutChangingEpisodeOrder() throws {
        let show = try decode(ShowDetail.self, #"{"title":"Тест","seasons":[{"id":1,"season_number":1,"name":"Первый","episodes":[]},{"id":3,"season_number":3,"name":"Третий","episodes":[{"id":31,"season_number":3,"episode_number":1,"name":"Начало"},{"id":32,"season_number":3,"episode_number":2,"name":"Продолжение"}]},{"id":2,"season_number":2,"name":"Второй","episodes":[]}]}"#)
        let seasons = try XCTUnwrap(show.seasons).newestFirst
        XCTAssertEqual(seasons.map(\.seasonNumber), [3, 2, 1])
        XCTAssertEqual(seasons[0].episodes.map(\.episodeNumber), [1, 2])
    }

    func testNotificationPayloadRemainsPlainText() throws {
        let feed = try decode(NotificationFeed.self, #"{"notifications":[{"id":1,"type":"episode_released","status":"sent","read":false,"payload":"Вышла новая серия","created_at":"2026-09-22T10:00:00.123Z"}]}"#)
        let item = try XCTUnwrap(feed.notifications.first)
        XCTAssertEqual(item.title, "Новая серия")
        XCTAssertEqual(item.payload, "Вышла новая серия")
        XCTAssertFalse(item.read)
        XCTAssertNotEqual(item.dateText, item.createdAt)
    }
}
