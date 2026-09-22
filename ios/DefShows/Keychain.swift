import Foundation
import Security

enum Keychain {
    private static func query(_ server: String) -> [String: Any] {
        [kSecClass as String: kSecClassGenericPassword,
         kSecAttrService as String: "defShows.session",
         kSecAttrAccount as String: server]
    }

    static func read(server: String) throws -> Data? {
        var attributes = query(server)
        attributes[kSecReturnData as String] = true
        attributes[kSecMatchLimit as String] = kSecMatchLimitOne
        var result: CFTypeRef?
        let status = SecItemCopyMatching(attributes as CFDictionary, &result)
        if status == errSecItemNotFound { return nil }
        guard status == errSecSuccess else { throw failure(status) }
        return result as? Data
    }

    static func save(_ data: Data, server: String) throws {
        let attributes: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly
        ]
        let status = SecItemUpdate(query(server) as CFDictionary, attributes as CFDictionary)
        if status == errSecItemNotFound {
            let insert = query(server).merging(attributes) { _, new in new }
            let inserted = SecItemAdd(insert as CFDictionary, nil)
            guard inserted == errSecSuccess else { throw failure(inserted) }
        } else if status != errSecSuccess { throw failure(status) }
    }

    static func delete(server: String) throws {
        let status = SecItemDelete(query(server) as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else { throw failure(status) }
    }

    private static func failure(_ status: OSStatus) -> APIError {
        APIError(message: "Не удалось получить доступ к Keychain (\(status)).")
    }
}
