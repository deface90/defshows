import SwiftUI

struct MainTabsView: View {
    var body: some View {
        TabView {
            LibraryView().tabItem { Label("Мои сериалы", systemImage: "tv") }
            NavigationStack { CatalogBrowseView(discover: false) }
                .tabItem { Label("Поиск", systemImage: "magnifyingglass") }
            NavigationStack { CatalogBrowseView(discover: true) }
                .tabItem { Label("Подбор", systemImage: "sparkles") }
            NavigationStack { NotificationsView() }
                .tabItem { Label("Уведомления", systemImage: "bell") }
            NavigationStack { MoreView() }
                .tabItem { Label("Ещё", systemImage: "ellipsis.circle") }
        }
    }
}

struct MoreView: View {
    @EnvironmentObject private var session: Session
    @State private var busy = false
    @State private var error: String?
    var body: some View {
        List {
            Section {
                NavigationLink { UsersView() } label: { Label("Пользователи", systemImage: "person.2") }
                NavigationLink { SettingsView() } label: { Label("Настройки", systemImage: "gearshape") }
            }
            Section {
                Button(role: .destructive) {
                    busy = true
                    Task {
                        defer { busy = false }
                        do { try await session.logout() }
                        catch { self.error = error.localizedDescription }
                    }
                } label: { Label("Выйти из аккаунта", systemImage: "rectangle.portrait.and.arrow.right") }
                .disabled(busy)
                if busy { ProgressView() }
                if let error { Text(error).foregroundStyle(.red) }
            }
        }.navigationTitle("Ещё")
    }
}

struct CatalogRow: View {
    let title: String
    let poster: String?
    var subtitle: String? = nil
    @EnvironmentObject private var session: Session
    var body: some View {
        HStack(spacing: 12) {
            AsyncImage(url: session.posterURL(poster)) { image in
                image.resizable().scaledToFill()
            } placeholder: {
                Rectangle().fill(.quaternary).overlay { Image(systemName: "tv").foregroundStyle(.secondary) }
            }.frame(width: 52, height: 78).clipShape(RoundedRectangle(cornerRadius: 8))
            VStack(alignment: .leading, spacing: 6) {
                Text(title).font(.headline)
                if let subtitle, !subtitle.isEmpty { Text(subtitle).font(.caption).foregroundStyle(.secondary) }
            }
        }.padding(.vertical, 3)
    }
}
