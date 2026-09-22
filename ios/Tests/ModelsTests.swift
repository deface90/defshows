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
        let show = try decode(ShowDetail.self, #"{"id":1,"title":"Тест","seasons":[{"id":1,"season_number":1,"name":"Первый","episodes":[]},{"id":3,"season_number":3,"name":"Третий","episodes":[{"id":31,"season_number":3,"episode_number":1,"name":"Начало"},{"id":32,"season_number":3,"episode_number":2,"name":"Продолжение"}]},{"id":2,"season_number":2,"name":"Второй","episodes":[]}]}"#)
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
    func testDetailMetadataAndExternalRatings() throws {
        let show = try decode(ShowDetail.self, #"{"id":7,"title":"Тест","original_title":"Test","poster_url":"/images/poster.jpg","first_air_date":"2020-01-01","next_episode_air_date":"2026-10-01","airing_status":"airing","vote_average":8.2,"vote_count":120,"genres":[{"id":1,"name":"Драма"}],"ratings":[{"source":"IMDb","value":"8.5","votes":300}],"imdb_url":"https://www.imdb.com/title/tt123/","wikipedia_url":"https://ru.wikipedia.org/wiki/Test"}"#)
        XCTAssertEqual(show.originalTitle, "Test")
        XCTAssertEqual(show.posterUrl, "/images/poster.jpg")
        XCTAssertEqual(show.genres?.first?.name, "Драма")
        XCTAssertEqual(show.ratings?.first?.votes, 300)
        XCTAssertEqual(show.nextEpisodeAirDate, "2026-10-01")
        XCTAssertNotNil(show.imdbUrl)
    }

    func testContinueWatchingUsesServerEpisodeID() throws {
        let progress = try decode(WatchProgress.self, #"{"watched":1,"total":3,"watched_episode_ids":[10],"next_unwatched_episode_id":12}"#)
        XCTAssertEqual(progress.nextUnwatchedEpisodeId, 12)
        XCTAssertEqual(progress.watchedEpisodeIds, [10])
    }

    func testAiredEpisodesExcludeFutureAndUnknownDates() throws {
        let today = Calendar.current.date(from: DateComponents(year: 2026, month: 9, day: 22))!
        let aired = try decode(Episode.self, #"{"id":1,"season_number":2,"episode_number":3,"name":"Тест","air_date":"2026-09-22"}"#)
        let upcoming = try decode(Episode.self, #"{"id":2,"season_number":2,"episode_number":4,"name":"Тест","air_date":"2026-09-23"}"#)
        let unknown = try decode(Episode.self, #"{"id":3,"season_number":2,"episode_number":5,"name":"Тест","air_date":null}"#)
        XCTAssertTrue(aired.hasAired(on: today))
        XCTAssertFalse(upcoming.hasAired(on: today))
        XCTAssertFalse(unknown.hasAired(on: today))
        XCTAssertEqual(aired.code, "S02E03")
    }

    func testPrivateNotesDecodeScopeAndMultilineBody() throws {
        let result = try decode(NoteList.self, #"{"notes":[{"id":3,"scope":"episode","season_number":2,"episode_number":4,"body":"Первая строка\nВторая строка"},{"id":4,"scope":"show","season_number":null,"episode_number":null,"body":"Общее"}]}"#)
        XCTAssertEqual(result.notes[0].scopeTitle, "S2E4")
        XCTAssertTrue(result.notes[0].body.contains("\n"))
        XCTAssertEqual(result.notes[1].scopeTitle, "Сериал")
    }

}
