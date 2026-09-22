import SwiftUI

@main
struct DefShowsApp: App {
    @StateObject private var session = Session()
    @State private var error: String?

    var body: some Scene {
        WindowGroup {
            Group {
                if session.signedIn { LibraryView() }
                else { LoginView() }
            }
            .environmentObject(session)
            .tint(.indigo)
            .task {
                do { try session.restore() }
                catch { self.error = error.localizedDescription }
            }
            .alert("Не удалось восстановить вход", isPresented: Binding(
                get: { error != nil }, set: { if !$0 { error = nil } }
            )) { Button("OK") { error = nil } } message: { Text(error ?? "") }
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
            }
            .disabled(busy)
            .navigationTitle("defShows")
        }
    }
}
