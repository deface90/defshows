import SwiftUI
import UserNotifications

@main
struct DefShowsApp: App {
    @UIApplicationDelegateAdaptor(NotificationDelegate.self) private var appDelegate
    @StateObject private var session = Session()
    @State private var error: String?

    var body: some Scene {
        WindowGroup {
            Group {
                if session.signedIn { MainTabsView() }
                else { LoginView() }
            }
            .environmentObject(session)
            .tint(.indigo)
            .task {
                do { try session.restore() }
                catch { self.error = error.localizedDescription }
                if session.signedIn { await requestPushAuthorization() }
            }
            .onChange(of: session.signedIn) { _, signedIn in
                if signedIn { Task { await requestPushAuthorization() } }
            }
            .onReceive(NotificationCenter.default.publisher(for: .apnsDeviceTokenRegistered)) { note in
                guard let token = note.object as? String else { return }
                Task { try? await session.registerDeviceToken(token) }
            }
            .alert("Не удалось восстановить вход", isPresented: Binding(
                get: { error != nil }, set: { if !$0 { error = nil } }
            )) { Button("OK") { error = nil } } message: { Text(error ?? "") }
        }
    }

    @MainActor
    private func requestPushAuthorization() async {
        let granted = (try? await UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge])) ?? false
        if granted {
            await UIApplication.shared.registerForRemoteNotifications()
        }
    }
}

struct LoginView: View {
    @EnvironmentObject private var session: Session
    @State private var email = ""
    @State private var password = ""
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    VStack(alignment: .leading, spacing: 10) {
                        Image(systemName: "play.tv.fill").font(.largeTitle).foregroundStyle(.indigo)
                        Text("Твои сериалы. Твой темп.").font(.title2.bold())
                        Text("Войди в существующий аккаунт defShows.").foregroundStyle(.secondary)
                    }.padding(.vertical)
                }
                Section("Аккаунт") {
                    TextField("Email", text: $email)
                        .keyboardType(.emailAddress).textContentType(.username)
                        .textInputAutocapitalization(.never).autocorrectionDisabled()
                    SecureField("Пароль", text: $password).textContentType(.password)
                }
                if let error { Section { Text(error).foregroundStyle(.red) } }
                Section {
                    Button {
                        busy = true
                        error = nil
                        Task {
                            defer { busy = false }
                            do {
                                try await session.login(email: email.trimmingCharacters(in: .whitespacesAndNewlines), password: password)
                                password = ""
                            } catch { self.error = error.localizedDescription }
                        }
                    } label: {
                        HStack {
                            Spacer()
                            if busy { ProgressView() } else { Text("Войти").bold() }
                            Spacer()
                        }
                    }
                    .disabled(email.isEmpty || password.isEmpty || busy)
                }
                Section {
                    NavigationLink("Нет аккаунта? Зарегистрироваться") { RegisterView() }
                }
                Section {
                    PrivacyPolicyLink()
                }
            }
            .disabled(busy)
            .navigationTitle("defShows")
        }
    }
}

struct RegisterView: View {
    @EnvironmentObject private var session: Session
    @State private var email = ""
    @State private var password = ""
    @State private var confirmPassword = ""
    @State private var busy = false
    @State private var error: String?

    private var passwordTooShort: Bool { !password.isEmpty && password.count < 6 }
    private var passwordsMismatch: Bool { !confirmPassword.isEmpty && password != confirmPassword }
    private var canSubmit: Bool {
        !email.isEmpty && password.count >= 6 && password == confirmPassword && !busy
    }

    var body: some View {
        Form {
            Section {
                VStack(alignment: .leading, spacing: 10) {
                    Image(systemName: "person.crop.circle.badge.plus").font(.largeTitle).foregroundStyle(.indigo)
                    Text("Создай аккаунт").font(.title2.bold())
                    Text("Сохраняй прогресс и синхронизируй его между устройствами.").foregroundStyle(.secondary)
                }.padding(.vertical)
            }
            Section("Аккаунт") {
                TextField("Email", text: $email)
                    .keyboardType(.emailAddress).textContentType(.username)
                    .textInputAutocapitalization(.never).autocorrectionDisabled()
                SecureField("Пароль", text: $password).textContentType(.newPassword)
                if passwordTooShort {
                    Text("Минимум 6 символов").font(.caption).foregroundStyle(.red)
                }
                SecureField("Повторите пароль", text: $confirmPassword).textContentType(.newPassword)
                if passwordsMismatch {
                    Text("Пароли не совпадают").font(.caption).foregroundStyle(.red)
                }
            }
            if let error { Section { Text(error).foregroundStyle(.red) } }
            Section {
                Button {
                    busy = true
                    error = nil
                    Task {
                        defer { busy = false }
                        do {
                            try await session.register(email: email.trimmingCharacters(in: .whitespacesAndNewlines), password: password)
                        } catch { self.error = error.localizedDescription }
                    }
                } label: {
                    HStack {
                        Spacer()
                        if busy { ProgressView() } else { Text("Зарегистрироваться").bold() }
                        Spacer()
                    }
                }
                .disabled(!canSubmit)
            }
            Section {
                PrivacyPolicyLink()
            }
        }
        .disabled(busy)
        .navigationTitle("Регистрация")
    }
}
