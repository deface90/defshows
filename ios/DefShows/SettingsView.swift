import SwiftUI

struct SettingsView: View {
    @EnvironmentObject private var session: Session
    @State private var settings: AccountSettings?
    @State private var preferences: NotificationPreferences?
    @State private var timezone = ""
    @State private var isPublic = false
    @State private var episodeRelease = false
    @State private var seasonStart = false
    @State private var weeklyDigest = false
    @State private var leadTimeHours = 0
    @State private var busy = false
    @State private var error: String?
    @State private var message: String?

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
                Section("Профиль") {
                    Toggle("Публичная коллекция", isOn: $isPublic)
                    Picker("Часовой пояс", selection: $timezone) {
                        ForEach(Array(Set(TimeZone.knownTimeZoneIdentifiers + [timezone])).sorted(), id: \.self) { zone in
                            Text(zone).tag(zone)
                        }
                    }
                    Button("Сохранить профиль") { Task { await saveProfile() } }
                }.disabled(busy)
            }
            if preferences != nil {
                Section {
                    Toggle("Выход новых серий", isOn: $episodeRelease)
                    Toggle("Начало сезона", isOn: $seasonStart)
                    Toggle("Еженедельная сводка", isOn: $weeklyDigest)
                    Stepper("За \(leadTimeHours) ч. до выхода", value: $leadTimeHours, in: 0...168)
                    Button("Сохранить уведомления") { Task { await savePreferences() } }
                } header: { Text("Уведомления сервиса") }
                  footer: { Text("Эти настройки действуют для твоего аккаунта. Push-уведомления iPhone пока не подключены.") }
                  .disabled(busy)
            }
        }
        .navigationTitle("Настройки")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            let account: AccountSettings = try await session.get("me/settings")
            settings = account
            timezone = account.timezone
            isPublic = account.isPublic
            let prefs: NotificationPreferences = try await session.get("me/notifications/prefs")
            preferences = prefs
            episodeRelease = prefs.episodeRelease
            seasonStart = prefs.seasonStart
            weeklyDigest = prefs.weeklyDigest
            leadTimeHours = prefs.leadTimeHours
            error = nil
        } catch { self.error = error.localizedDescription }
    }

    private func saveProfile() async {
        guard !busy else { return }
        busy = true
        error = nil
        message = nil
        defer { busy = false }
        do {
            try await session.mutate("me/settings", method: "PATCH", body: ["timezone": timezone, "is_public": isPublic])
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
                "weekly_digest": weeklyDigest, "lead_time_hours": leadTimeHours
            ])
            message = "Настройки уведомлений сохранены."
        } catch { self.error = error.localizedDescription }
    }
}
