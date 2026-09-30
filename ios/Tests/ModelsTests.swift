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

    func testSeasonToExpandFollowsTheEarliestUnwatchedEpisode() throws {
        let show = try decode(ShowDetail.self, #"{"id":1,"title":"Тест","seasons":[{"id":1,"season_number":1,"name":"Первый","episodes":[{"id":11,"season_number":1,"episode_number":1,"name":"Начало"}]},{"id":2,"season_number":2,"name":"Второй","episodes":[{"id":21,"season_number":2,"episode_number":1,"name":"Второе"}]},{"id":3,"season_number":3,"name":"Третий","episodes":[{"id":31,"season_number":3,"episode_number":1,"name":"Третье"}]}]}"#)
        let seasons = try XCTUnwrap(show.seasons)
        XCTAssertEqual(seasons.seasonToExpand(nextUnwatchedEpisodeID: 21)?.seasonNumber, 2)
        // Nothing left to watch, an unknown episode, or an untracked show: newest season.
        XCTAssertEqual(seasons.seasonToExpand(nextUnwatchedEpisodeID: nil)?.seasonNumber, 3)
        XCTAssertEqual(seasons.seasonToExpand(nextUnwatchedEpisodeID: 999)?.seasonNumber, 3)
        XCTAssertNil([Season]().seasonToExpand(nextUnwatchedEpisodeID: 21))
    }

    func testNotificationPayloadRemainsPlainText() throws {
        let feed = try decode(NotificationFeed.self, #"{"notifications":[{"id":1,"type":"episode_released","status":"sent","read":false,"payload":"Вышла новая серия","created_at":"2026-09-22T10:00:00.123Z"}]}"#)
        let item = try XCTUnwrap(feed.notifications.first)
        XCTAssertEqual(item.title, "Новая серия")
        XCTAssertEqual(item.payload, "Вышла новая серия")
        XCTAssertFalse(item.read)
        XCTAssertNotEqual(item.dateText, item.createdAt)
    }

    func testSeasonFinaleNotificationTitle() throws {
        let feed = try decode(NotificationFeed.self, #"{"notifications":[{"id":2,"type":"season_finale","status":"sent","read":true,"payload":"Сезон завершён","created_at":"2026-09-22T10:00:00.123Z"}]}"#)
        XCTAssertEqual(feed.notifications.first?.title, "Финал сезона")
    }

    func testStatsDecodeAndWatchTimeFormatting() throws {
        let stats = try decode(Stats.self, #"{"shows_tracked":12,"shows_completed":4,"seasons_watched":9,"episodes_watched":120,"minutes_watched":3600}"#)
        XCTAssertEqual(stats.showsTracked, 12)
        XCTAssertEqual(stats.episodesWatched, 120)
        // 3600 minutes = 2 days 12 hours.
        XCTAssertEqual(stats.watchTimeText, "2 д 12 ч")
        XCTAssertEqual(Stats(showsTracked: 0, showsCompleted: 0, seasonsWatched: 0, episodesWatched: 0, minutesWatched: 90).watchTimeText, "1 ч 30 мин")
        XCTAssertEqual(Stats(showsTracked: 0, showsCompleted: 0, seasonsWatched: 0, episodesWatched: 0, minutesWatched: 0).watchTimeText, "0 мин")
    }

    func testWatchProgressDecodesNewSeasonAndUnwatchedWhenPresent() throws {
        let progress = try decode(WatchProgress.self, #"{"watched":8,"total":16,"watched_episode_ids":[1],"next_unwatched_episode_id":9,"unwatched":8,"new_full_season":2}"#)
        XCTAssertEqual(progress.unwatched, 8)
        XCTAssertEqual(progress.newFullSeason, 2)
        // Absent fields stay nil (older payloads / caught-up shows).
        let plain = try decode(WatchProgress.self, #"{"watched":1,"total":3,"watched_episode_ids":[10],"next_unwatched_episode_id":12}"#)
        XCTAssertNil(plain.unwatched)
        XCTAssertNil(plain.newFullSeason)
    }

    func testShowReferenceExposesDistinctOriginalTitleAndAiring() throws {
        let both = try decode(TrackedList.self, #"{"tracked":[{"show":{"id":1,"tmdb_id":42,"title":"Тест","original_title":"Test","airing_status":"airing"},"user_show":{"status":"watching","favorite":false},"progress":{"watched":1,"total":2,"watched_episode_ids":[1],"next_unwatched_episode_id":2}}]}"#)
        let show = try XCTUnwrap(both.tracked.first?.show)
        XCTAssertEqual(show.distinctOriginalTitle, "Test")
        XCTAssertEqual(show.airingStatusTitle, "Идёт")
        // Original identical to the localized title is hidden.
        let same = try decode(ShowReference.self, #"{"id":1,"tmdb_id":42,"title":"Тест","original_title":"Тест"}"#)
        XCTAssertNil(same.distinctOriginalTitle)
        XCTAssertNil(same.airingStatusTitle)
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
