import SwiftUI

private struct ShowSheetItem: Identifiable {
    let id: Int
}

struct MainTabsView: View {
    @EnvironmentObject private var session: Session
    @State private var openShow: ShowSheetItem?

    var body: some View {
        TabView {
            LibraryView().tabItem { Label("Мои сериалы", systemImage: "tv") }
            NavigationStack { FeedView() }
                .tabItem { Label("Лента", systemImage: "square.stack.3d.up") }
            NavigationStack { BrowseView() }
                .tabItem { Label("Обзор", systemImage: "magnifyingglass") }
            NavigationStack { PeopleView() }
                .tabItem { Label("Люди", systemImage: "person.2") }
                .badge(session.pendingFollowRequests)
            NavigationStack { MoreView() }
                .tabItem { Label("Ещё", systemImage: "ellipsis.circle") }
        }
        .task { await session.refreshPendingRequests() }
        .onReceive(NotificationCenter.default.publisher(for: .pushOpenShow)) { note in
            guard let showID = note.object as? Int else { return }
            openShow = ShowSheetItem(id: showID)
        }
        .sheet(item: $openShow) { item in
            NavigationStack { ShowView(showID: item.id) }
        }
    }
}

/// BrowseView hosts catalog search and discovery under one tab, switched by a segmented
/// control. Each mode carries its own identity via `.id`, so toggling starts the other
/// mode fresh rather than leaking one mode's query/results into the other.
struct BrowseView: View {
    private enum Mode: Hashable { case search, discover }
    @State private var mode: Mode = .search

    var body: some View {
        VStack(spacing: 0) {
            Picker("", selection: $mode) {
                Text("Поиск").tag(Mode.search)
                Text("Подбор").tag(Mode.discover)
            }
            .pickerStyle(.segmented)
            .padding([.horizontal, .top])
            CatalogBrowseView(discover: mode == .discover)
                .id(mode)
        }
        .navigationTitle("Обзор")
        .navigationBarTitleDisplayMode(.inline)
    }
}

/// PeopleView is the social hub: the follows/followers/requests graph, with a toolbar
/// shortcut to search the user directory.
struct PeopleView: View {
    @State private var findingUsers = false

    var body: some View {
        FollowsView()
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { findingUsers = true } label: {
                        Image(systemName: "person.badge.plus")
                    }
                    .accessibilityLabel("Найти пользователей")
                }
            }
            .sheet(isPresented: $findingUsers) {
                NavigationStack {
                    UsersView()
                        .toolbar {
                            ToolbarItem(placement: .topBarLeading) {
                                Button("Готово") { findingUsers = false }
                            }
                        }
                }
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
                NavigationLink { SettingsView() } label: { Label("Настройки", systemImage: "gearshape") }
            }
            Section("О сервисе") {
                PrivacyPolicyLink()
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

struct PrivacyPolicyLink: View {
    private static let destination = URL(string: "https://shows.deface.dev/privacy")!

    var body: some View {
        Link(destination: Self.destination) {
            Label("Политика конфиденциальности", systemImage: "hand.raised")
        }
        .accessibilityHint("Открывает политику конфиденциальности в браузере")
    }
}
