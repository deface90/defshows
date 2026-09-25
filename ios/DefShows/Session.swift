import Foundation
import Combine

@MainActor
final class Session: ObservableObject {
    @Published private(set) var signedIn = false
    @Published private(set) var collectionRevision = 0
    // Public API prefix from the deployed web client configuration.
    private let server = "https://shows.deface.dev/api"
    private var tokens: Tokens?
    private var refreshTask: Task<Tokens, Error>?
    private let decoder: JSONDecoder = {
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return decoder
    }()
    private let transport: URLSession = {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.timeoutIntervalForRequest = 30
        return URLSession(configuration: configuration)
    }()

    func restore() throws {
        guard let data = try Keychain.read(server: server) else { return }
        tokens = try JSONDecoder().decode(Tokens.self, from: data)
        signedIn = true
    }

    func login(email: String, password: String) async throws {
        let data = try await send("auth/login", method: "POST", body: ["email": email, "password": password])
        let response = try decoder.decode(AuthResponse.self, from: data)
        try persist(response.tokens)
        signedIn = true
    }

    func register(email: String, password: String) async throws {
        do {
            let data = try await send("auth/register", method: "POST", body: ["email": email, "password": password])
            let response = try decoder.decode(AuthResponse.self, from: data)
            try persist(response.tokens)
            signedIn = true
        } catch let error as HTTPFailure where error.status == 409 {
            // The server's message is English ("email already registered"); the
            // app is Russian-only, so localize this one expected case ourselves.
            throw APIError(message: "Этот email уже зарегистрирован")
        }
    }

    func get<T: Decodable>(_ path: String, query: [URLQueryItem] = []) async throws -> T {
        let data = try await authorized(path, query: query)
        return try decoder.decode(T.self, from: data)
    }

    func mutate(_ path: String, method: String = "POST", body: [String: Any]? = nil) async throws {
        _ = try await authorized(path, method: method, body: body)
        if path.hasPrefix("me/shows") { collectionDidChange() }
    }

    func addShow(tmdbID: Int) async throws {
        try await mutate("me/shows", body: ["tmdb_id": tmdbID])
    }

    func collectionDidChange() { collectionRevision += 1 }

    func setWatched(showID: Int, episodeID: Int, watched: Bool, notifyCollection: Bool = true) async throws {
        _ = try await authorized("me/shows/\(showID)/episodes/\(episodeID)/watch", method: watched ? "POST" : "DELETE")
        if notifyCollection { collectionDidChange() }
    }

    func logout() async throws {
        // Best-effort: don't let a failed unregister block logout.
        try? await unregisterDeviceToken()
        // Keep the local session if revocation fails so logout can be retried.
        if let tokens {
            _ = try await send("auth/logout", method: "POST", body: ["refresh_token": tokens.refreshToken])
        }
        try clear()
    }

    /// Permanently deletes the account and every piece of data tied to it on the server.
    func deleteAccount() async throws {
        _ = try await authorized("auth/me", method: "DELETE")
        try clear()
    }

    func registerDeviceToken(_ token: String) async throws {
        try await mutate("me/device-token", body: ["token": token, "platform": "ios"])
    }

    func unregisterDeviceToken() async throws {
        try await mutate("me/device-token", method: "DELETE")
    }

    func posterURL(_ source: String?) -> URL? {
        guard let source, !source.isEmpty, let base = URL(string: server + "/") else { return nil }
        if let url = URL(string: source), url.host == "image.tmdb.org" {
            let parts = url.pathComponents
            if parts.count == 5, parts[1] == "t", parts[2] == "p" {
                return base.appendingPathComponent("images/tmdb/\(parts[3])/\(parts[4])")
            }
        }
        return URL(string: source, relativeTo: base)?.absoluteURL
    }

    private func authorized(_ path: String, method: String = "GET", body: [String: Any]? = nil, query: [URLQueryItem] = []) async throws -> Data {
        guard let current = tokens else { throw APIError(message: "Войди в аккаунт.") }
        do {
            return try await send(path, method: method, body: body, access: current.accessToken, query: query)
        } catch let error as HTTPFailure where error.status == 401 {
            // Another request may already have refreshed the same expired token.
            if tokens?.accessToken == current.accessToken { try await refresh() }
            guard let renewed = tokens else { throw APIError(message: "Войди в аккаунт заново.") }
            do {
                return try await send(path, method: method, body: body, access: renewed.accessToken, query: query)
            } catch let retry as HTTPFailure where retry.status == 401 {
                try clear()
                throw APIError(message: "Сессия истекла. Войди заново.")
            }
        }
    }

    private func refresh() async throws {
        if let refreshTask { _ = try await refreshTask.value; return }
        guard let current = tokens else { throw APIError(message: "Войди в аккаунт.") }
        let task = Task { () throws -> Tokens in
            let data = try await self.send("auth/refresh", method: "POST", body: ["refresh_token": current.refreshToken])
            let renewed = try self.decoder.decode(Tokens.self, from: data)
            try self.persist(renewed)
            return renewed
        }
        refreshTask = task
        defer { refreshTask = nil }
        do { _ = try await task.value }
        catch let failure as HTTPFailure where failure.status == 401 {
            try clear()
            throw APIError(message: "Сессия истекла. Войди заново.")
        }
    }

    private func persist(_ value: Tokens) throws {
        try Keychain.save(JSONEncoder().encode(value), server: server)
        tokens = value
    }

    private func clear() throws {
        try Keychain.delete(server: server)
        tokens = nil
        signedIn = false
    }

    private func send(_ path: String, method: String = "GET", body: [String: Any]? = nil, access: String? = nil, query: [URLQueryItem] = []) async throws -> Data {
        guard let base = URL(string: server + "/") else { throw APIError(message: "Некорректный адрес API.") }
        guard var components = URLComponents(url: base.appendingPathComponent(path), resolvingAgainstBaseURL: false) else {
            throw APIError(message: "Некорректный адрес запроса.")
        }
        if !query.isEmpty { components.queryItems = query }
        guard let url = components.url else { throw APIError(message: "Некорректный адрес запроса.") }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let access { request.setValue("Bearer \(access)", forHTTPHeaderField: "Authorization") }
        if let body {
            request.httpBody = try JSONSerialization.data(withJSONObject: body)
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        let (data, response) = try await transport.data(for: request)
        guard let response = response as? HTTPURLResponse else { throw APIError(message: "Сервер не ответил.") }
        guard (200..<300).contains(response.statusCode) else {
            struct ErrorBody: Decodable { let message: String }
            let message = (try? decoder.decode(ErrorBody.self, from: data))?.message
            throw HTTPFailure(status: response.statusCode, message: message ?? "Ошибка сервера (\(response.statusCode)).")
        }
        return data
    }
}

private struct HTTPFailure: LocalizedError {
    let status: Int
    let message: String
    var errorDescription: String? { message }
}
