// swift-tools-version: 5.9
import PackageDescription

// Model tests can run on a Mac without booting an iOS simulator.
let package = Package(
    name: "DefShowsModels",
    platforms: [.macOS(.v13)],
    products: [.library(name: "DefShowsModels", targets: ["DefShowsModels"])],
    targets: [
        .target(name: "DefShowsModels", path: "DefShows", exclude: [
            "Assets.xcassets", "PushNotifications.swift", "DefShows.entitlements",
            "DefShowsApp.swift", "Keychain.swift", "Session.swift", "LibraryView.swift",
            "ShowView.swift", "ShowInformation.swift", "ShowNotesView.swift", "MainTabsView.swift", "CatalogBrowseView.swift",
            "NotificationsView.swift", "UsersView.swift", "SettingsView.swift"
        ], sources: ["Models.swift", "BrowseModels.swift"]),
        .testTarget(name: "DefShowsModelsTests", dependencies: ["DefShowsModels"], path: "Tests")
    ]
)
