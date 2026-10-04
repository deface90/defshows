import SwiftUI

struct SettingsView: View {
    @EnvironmentObject private var session: Session
    @State private var settings: AccountSettings?
    @State private var preferences: NotificationPreferences?
    @State private var timezone = ""
    @State private var isPublic = false
    @State private var displayName = ""
    @State private var episodeRelease = false
    @State private var seasonStart = false
    @State private var seasonFinale = false
    @State private var weeklyDigest = false
    @State private var leadTimeHours = 0
    @State private var busy = false
    @State private var error: String?
    @State private var message: String?
    @State private var confirmingDeletion = false

    var body: some View {
        Form {
            if let error {
                Section {
                    Text(error).foregroundStyle(.red)
                    if settings == nil || preferences == nil {
                        Button("Повторить загрузку") { Task { await load() } }.disabled(busy)
                    }
                }
            }
            if let message { Section { Text(message).foregroundStyle(.secondary) } }
            if busy { ProgressView() }
            if settings != nil {
                Section {
                    TextField("Отображаемое имя", text: $displayName)
                        .textInputAutocapitalization(.words)
                    Toggle("Публичная коллекция", isOn: $isPublic)
                    Picker("Часовой пояс", selection: $timezone) {
                        ForEach(Array(Set(TimeZone.knownTimeZoneIdentifiers + [timezone])).sorted(), id: \.self) { zone in
                            Text(zone).tag(zone)
                        }
                    }
                    Button("Сохранить профиль") { Task { await saveProfile() } }
                        .disabled(!displayNameValid)
                } header: {
                    Text("Профиль")
                } footer: {
                    Text("Имя видят другие пользователи. Пусто — вернётся имя по умолчанию. До 50 символов.")
                }.disabled(busy)
                BlockedUsersSection()
            }
            if preferences != nil {
                Section {
                    Toggle("Выход новых серий", isOn: $episodeRelease)
                    Toggle("Начало сезона", isOn: $seasonStart)
                    Toggle("Финал сезона", isOn: $seasonFinale)
                    Toggle("Еженедельная сводка", isOn: $weeklyDigest)
                    Stepper("За \(leadTimeHours) ч. до выхода", value: $leadTimeHours, in: 0...168)
                    Button("Сохранить уведомления") { Task { await savePreferences() } }
                } header: { Text("Уведомления сервиса") }
                  footer: { Text("Эти настройки действуют для твоего аккаунта: push-уведомлений на iPhone и Telegram. Разрешение на push можно отключить в настройках iOS. «Финал сезона» приходит по факту выхода последней серии сезона — настройка «за сколько часов» на него не влияет.") }
                  .disabled(busy)
            }
            Section {
                Button("Удалить аккаунт", role: .destructive) { confirmingDeletion = true }
                    .disabled(busy)
            } footer: {
                Text("Аккаунт и все данные — коллекция, отметки просмотра, заметки, уведомления — будут удалены без возможности восстановления.")
            }
        }
        .confirmationDialog("Удалить аккаунт?", isPresented: $confirmingDeletion, titleVisibility: .visible) {
            Button("Удалить навсегда", role: .destructive) { Task { await deleteAccount() } }
            Button("Отмена", role: .cancel) {}
        } message: {
            Text("Это действие нельзя отменить.")
        }
        .navigationTitle("Настройки")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private var displayNameValid: Bool { AccountSettings.isValidDisplayName(displayName) }

    private func load() async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            let account: AccountSettings = try await session.get("me/settings")
            settings = account
            timezone = account.timezone
            isPublic = account.isPublic
            displayName = account.displayName
            let prefs: NotificationPreferences = try await session.get("me/notifications/prefs")
            preferences = prefs
            episodeRelease = prefs.episodeRelease
            seasonStart = prefs.seasonStart
            seasonFinale = prefs.seasonFinale
            weeklyDigest = prefs.weeklyDigest
            leadTimeHours = prefs.leadTimeHours
            error = nil
        } catch { self.error = error.localizedDescription }
    }

    private func deleteAccount() async {
        guard !busy else { return }
        busy = true
        error = nil
        message = nil
        defer { busy = false }
        // Best-effort: the server row and its token go away with the account anyway.
        try? await session.unregisterDeviceToken()
        do { try await session.deleteAccount() } catch { self.error = error.localizedDescription }
    }

    private func saveProfile() async {
        guard !busy else { return }
        busy = true
        error = nil
        message = nil
        defer { busy = false }
        do {
            // The server echoes the effective display name (derived default when blank),
            // so reload from the PATCH response to reflect any fallback.
            let updated: AccountSettings = try await session.patch("me/settings", body: [
                "timezone": timezone, "is_public": isPublic,
                "display_name": displayName.trimmingCharacters(in: .whitespacesAndNewlines)
            ])
            settings = updated
            displayName = updated.displayName
            message = "Настройки профиля сохранены."
        } catch { self.error = error.localizedDescription }
    }

    private func savePreferences() async {
        guard !busy else { return }
        busy = true
        error = nil
        message = nil
        defer { busy = false }
        do {
            try await session.mutate("me/notifications/prefs", method: "PATCH", body: [
                "episode_release": episodeRelease, "season_start": seasonStart,
                "season_finale": seasonFinale, "weekly_digest": weeklyDigest,
                "lead_time_hours": leadTimeHours
            ])
            message = "Настройки уведомлений сохранены."
        } catch { self.error = error.localizedDescription }
    }
}

/// Lists the users the current user has blocked (GET /me/blocks) with an Unblock action.
/// Paginated: "Показать ещё" loads older blocks so none become unreachable.
private struct BlockedUsersSection: View {
    @EnvironmentObject private var session: Session
    private static let pageSize = 20
    @State private var users: [FollowUser] = []
    @State private var total = 0
    @State private var page = 0 // number of pages loaded so far
    @State private var loaded = false
    @State private var busy = false
    @State private var error: String?

    private var hasMore: Bool { users.count < total }

    var body: some View {
        Section {
            if let error { Text(error).foregroundStyle(.red) }
            if loaded && users.isEmpty {
                Text("Нет заблокированных пользователей").foregroundStyle(.secondary)
            }
            ForEach(users) { u in
                HStack {
                    Text(u.displayName)
                    Spacer()
                    Button("Разблокировать") { Task { await unblock(u) } }
                        .buttonStyle(.bordered).controlSize(.small).disabled(busy)
                }
            }
            if hasMore {
                Button("Показать ещё") { Task { await loadMore() } }.disabled(busy)
            }
        } header: {
            Text("Заблокированные")
        }
        .task { await load() }
    }

    private func load() async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            let list = try await session.blocks(page: 1, pageSize: Self.pageSize)
            users = list.users
            total = list.total
            page = 1
            loaded = true
            error = nil
        } catch { self.error = error.localizedDescription }
    }

    private func loadMore() async {
        guard !busy, hasMore else { return }
        busy = true
        defer { busy = false }
        do {
            let next = page + 1
            let list = try await session.blocks(page: next, pageSize: Self.pageSize)
            users.append(contentsOf: list.users)
            total = list.total
            page = next
            error = nil
        } catch { self.error = error.localizedDescription }
    }

    private func unblock(_ user: FollowUser) async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            try await session.unblock(userID: user.id)
            // Removing a row shifts every later block up by one slot in the offset-paginated
            // result, so a naive local remove would make the next "Показать ещё" (page+1)
            // skip the row that slid into the previous page's last position. Refetch all the
            // currently-loaded pages (1..page) from a consistent offset instead.
            let pagesLoaded = max(page, 1)
            var refreshed: [FollowUser] = []
            var newTotal = total
            for p in 1...pagesLoaded {
                let list = try await session.blocks(page: p, pageSize: Self.pageSize)
                refreshed.append(contentsOf: list.users)
                newTotal = list.total
            }
            users = refreshed
            total = newTotal
            // The last page may have emptied out after the removal; keep `page` consistent
            // with how many pages actually returned rows so hasMore/loadMore stay correct.
            page = max(1, Int((Double(refreshed.count) / Double(Self.pageSize)).rounded(.up)))
            error = nil
        } catch { self.error = error.localizedDescription }
    }
}
