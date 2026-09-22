import SwiftUI

struct ShowInformation: View {
    let show: ShowDetail
    @EnvironmentObject private var session: Session

    private var airingTitle: String {
        switch show.airingStatus {
        case "not_started": return "Ещё не начался"
        case "airing": return "Идёт"
        case "between_seasons": return "Между сезонами"
        case "ended": return "Завершён"
        default: return show.airingStatus ?? ""
        }
    }

    var body: some View {
        Section {
            VStack(alignment: .leading, spacing: 14) {
                HStack {
                    Spacer()
                    AsyncImage(url: session.posterURL(show.posterUrl)) { image in
                        image.resizable().scaledToFill()
                    } placeholder: {
                        Rectangle().fill(.quaternary)
                            .overlay { Image(systemName: "tv").font(.largeTitle).foregroundStyle(.secondary) }
                    }
                    .frame(width: 160, height: 240)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                    .accessibilityLabel("Постер: \(show.title)")
                    Spacer()
                }
                Text(show.title).font(.title2.bold())
                if let original = show.originalTitle, original != show.title, !original.isEmpty {
                    Text(original).font(.headline).foregroundStyle(.secondary)
                }
                Text([airingTitle, show.firstAirDate.map { String($0.prefix(4)) } ?? ""]
                    .filter { !$0.isEmpty }.joined(separator: " · "))
                    .font(.subheadline).foregroundStyle(.secondary)
                if let genres = show.genres, !genres.isEmpty {
                    Text(genres.map(\.name).joined(separator: " · ")).font(.subheadline)
                }
                if let next = show.nextEpisodeAirDate {
                    Label("Следующая серия: \(next)", systemImage: "calendar").font(.subheadline).foregroundStyle(.indigo)
                }
                if let rating = show.voteAverage, rating > 0 {
                    RatingLine(source: "TMDB", value: String(format: "%.1f / 10", rating), votes: show.voteCount)
                }
                ForEach(Array((show.ratings ?? []).enumerated()), id: \.offset) { _, rating in
                    RatingLine(source: rating.source, value: rating.value, votes: rating.votes)
                }
            }.padding(.vertical, 8)
        }
        Section("О сериале") {
            Text(show.overview.flatMap { $0.isEmpty ? nil : $0 } ?? "Описание отсутствует")
            if let url = externalURL(show.imdbUrl) { Link("IMDb ↗", destination: url) }
            if let url = externalURL(show.wikipediaUrl) { Link("Wikipedia ↗", destination: url) }
        }
    }

    private func externalURL(_ value: String?) -> URL? {
        guard let value, let url = URL(string: value), ["https", "http"].contains(url.scheme ?? "") else { return nil }
        return url
    }
}

private struct RatingLine: View {
    let source: String
    let value: String
    let votes: Int?
    var body: some View {
        HStack {
            Label("\(source): \(value)", systemImage: "star.fill").foregroundStyle(.orange)
            if let votes, votes > 0 { Text("\(votes) голосов").foregroundStyle(.secondary) }
        }.font(.caption)
    }
}

struct EpisodeInformation: View {
    let episode: Episode
    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text("\(episode.code) · \(episode.name)").foregroundStyle(.primary)
            HStack {
                if let date = episode.airDate {
                    Text(date).foregroundStyle(episode.hasAired() ? Color.secondary : Color.indigo)
                } else { Text("Дата выхода неизвестна").foregroundStyle(.secondary) }
                if let runtime = episode.runtime, runtime > 0 { Text("\(runtime) мин").foregroundStyle(.secondary) }
            }.font(.caption)
            if let rating = episode.voteAverage, let votes = episode.voteCount, votes > 0 {
                RatingLine(source: "TMDB", value: String(format: "%.1f", rating), votes: votes)
            }
        }.padding(.vertical, 3)
    }
}
