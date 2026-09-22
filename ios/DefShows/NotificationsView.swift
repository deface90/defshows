import SwiftUI

struct NotificationsView: View {
    @EnvironmentObject private var session: Session
    @State private var items: [FeedItem] = []
    @State private var loaded = false
    @State private var busy = false
    @State private var changing = false
    @State private var error: String?

    var body: some View {
        List {
            if let error {
                Text(error).foregroundStyle(.red)
                Button("Повторить") { Task { await load() } }.disabled(busy || changing)
            }
            if busy { ProgressView() }
            if loaded && items.isEmpty {
                ContentUnavailableView("Пока тихо", systemImage: "bell", description: Text("Здесь появятся уведомления о твоих сериалах."))
            }
            ForEach(items) { item in
                VStack(alignment: .leading, spacing: 8) {
                    HStack {
                        if !item.read { Circle().fill(.indigo).frame(width: 8, height: 8).accessibilityLabel("Новое") }
                        Text(item.title).font(.headline)
                    }
                    Text(item.payload)
                    Text(item.dateText).font(.caption).foregroundStyle(.secondary)
                    if !item.read {
                        Button("Отметить прочитанным") { Task { await markRead(id: item.id) } }
                            .font(.caption).disabled(busy || changing)
                    }
                }.padding(.vertical, 4)
            }
        }
        .navigationTitle("Уведомления")
        .toolbar {
            Button("Прочитать всё") { Task { await markRead(id: nil) } }
                .disabled(busy || changing || !items.contains { !$0.read })
        }
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        guard !busy, !changing else { return }
        busy = true
        defer { busy = false }
        do {
            let result: NotificationFeed = try await session.get("me/notifications")
            items = result.notifications
            loaded = true
            error = nil
        } catch { self.error = error.localizedDescription }
    }

    private func markRead(id: Int?) async {
        guard !busy, !changing else { return }
        changing = true
        defer { changing = false }
        do {
            let path = id.map { "me/notifications/\($0)/read" } ?? "me/notifications/read-all"
            try await session.mutate(path)
            for index in items.indices where id == nil || items[index].id == id { items[index].read = true }
            error = nil
        } catch { self.error = error.localizedDescription }
    }
}
