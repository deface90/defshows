import SwiftUI

struct ShowNotesView: View {
    let showID: Int
    let seasons: [Season]
    @EnvironmentObject private var session: Session
    @State private var notes: [ShowNote] = []
    @State private var loaded = false
    @State private var busy = false
    @State private var error: String?
    @State private var editor: NoteDraft?
    @State private var deleting: ShowNote?

    var body: some View {
        List {
            if let error {
                Text(error).foregroundStyle(.red)
                Button("Повторить") { Task { await load() } }.disabled(busy)
            }
            if busy { ProgressView() }
            if loaded && notes.isEmpty {
                ContentUnavailableView("Заметок пока нет", systemImage: "note.text", description: Text("Сохрани мысли о сериале, сезоне или отдельной серии. Заметки видны только тебе."))
            }
            ForEach(notes) { note in
                VStack(alignment: .leading, spacing: 10) {
                    Text(note.scopeTitle).font(.caption.bold()).foregroundStyle(.indigo)
                    Text(note.body).textSelection(.enabled)
                    HStack {
                        Button("Изменить") { editor = NoteDraft(note: note) }
                        Spacer()
                        Button("Удалить", role: .destructive) { deleting = note }
                    }.font(.caption).buttonStyle(.borderless).disabled(busy)
                }.padding(.vertical, 5)
            }
        }
        .navigationTitle("Мои заметки")
        .toolbar { Button { editor = NoteDraft() } label: { Image(systemName: "square.and.pencil") }.accessibilityLabel("Добавить заметку").disabled(busy) }
        .task { await load() }
        .refreshable { await load() }
        .sheet(item: $editor) { draft in
            NoteEditorView(showID: showID, seasons: seasons, draft: draft) {
                Task { await load() }
            }
        }
        .confirmationDialog("Удалить заметку?", isPresented: Binding(
            get: { deleting != nil }, set: { if !$0 { deleting = nil } }
        ), titleVisibility: .visible) {
            Button("Удалить", role: .destructive) {
                if let note = deleting { Task { await remove(note) } }
            }
        }
    }

    private func load() async {
        guard !busy else { return }
        busy = true
        defer { busy = false }
        do {
            let result: NoteList = try await session.get("me/notes", query: [URLQueryItem(name: "show_id", value: String(showID))])
            notes = result.notes
            loaded = true
            error = nil
        } catch { self.error = error.localizedDescription }
    }

    private func remove(_ note: ShowNote) async {
        guard !busy else { return }
        busy = true
        defer { busy = false; deleting = nil }
        do {
            try await session.mutate("me/notes/\(note.id)", method: "DELETE")
            notes.removeAll { $0.id == note.id }
            error = nil
        } catch { self.error = error.localizedDescription }
    }
}

private struct NoteDraft: Identifiable {
    let id = UUID()
    var noteID: Int? = nil
    var scope = "show"
    var seasonNumber = 1
    var episodeNumber = 1
    var body = ""

    init() {}
    init(note: ShowNote) {
        noteID = note.id
        scope = note.scope
        seasonNumber = note.seasonNumber ?? 1
        episodeNumber = note.episodeNumber ?? 1
        body = note.body
    }
}

private struct NoteEditorView: View {
    let showID: Int
    let seasons: [Season]
    @State var draft: NoteDraft
    let onSave: () -> Void
    @EnvironmentObject private var session: Session
    @Environment(\.dismiss) private var dismiss
    @State private var busy = false
    @State private var error: String?

    private var availableSeasons: [Season] { seasons.filter { $0.seasonNumber > 0 }.newestFirst }
    private var availableEpisodes: [Episode] {
        seasons.first { $0.seasonNumber == draft.seasonNumber }?.episodes.sorted { $0.episodeNumber < $1.episodeNumber } ?? []
    }
    private var valid: Bool {
        guard !draft.body.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return false }
        if draft.noteID != nil || draft.scope == "show" { return true }
        guard availableSeasons.contains(where: { $0.seasonNumber == draft.seasonNumber }) else { return false }
        return draft.scope != "episode" || availableEpisodes.contains { $0.episodeNumber == draft.episodeNumber }
    }

    var body: some View {
        NavigationStack {
            Form {
                if draft.noteID == nil {
                    Picker("Заметка к", selection: $draft.scope) {
                        Text("Сериалу").tag("show")
                        Text("Сезону").tag("season")
                        Text("Серии").tag("episode")
                    }
                    if draft.scope != "show" {
                        if availableSeasons.isEmpty { Text("Сезоны пока не загружены").foregroundStyle(.secondary) }
                        Picker("Сезон", selection: $draft.seasonNumber) {
                            ForEach(availableSeasons) { Text("Сезон \($0.seasonNumber)").tag($0.seasonNumber) }
                        }
                    }
                    if draft.scope == "episode" {
                        Picker("Серия", selection: $draft.episodeNumber) {
                            ForEach(availableEpisodes) { Text("\($0.episodeNumber). \($0.name)").tag($0.episodeNumber) }
                        }
                    }
                }
                Section("Текст заметки") {
                    TextEditor(text: $draft.body).frame(minHeight: 180).accessibilityLabel("Текст заметки")
                }
                if let error { Text(error).foregroundStyle(.red) }
                if busy { ProgressView() }
            }
            .disabled(busy)
            .navigationTitle(draft.noteID == nil ? "Новая заметка" : "Изменить заметку")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Отмена") { dismiss() }.disabled(busy) }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Сохранить") { Task { await save() } }.disabled(busy || !valid)
                }
            }
            .onAppear {
                if draft.noteID == nil, !availableSeasons.contains(where: { $0.seasonNumber == draft.seasonNumber }) {
                    draft.seasonNumber = availableSeasons.first?.seasonNumber ?? 1
                }
                if draft.noteID == nil, !availableEpisodes.contains(where: { $0.episodeNumber == draft.episodeNumber }) {
                    draft.episodeNumber = availableEpisodes.first?.episodeNumber ?? 1
                }
            }
            .onChange(of: draft.seasonNumber) { _, _ in
                draft.episodeNumber = availableEpisodes.first?.episodeNumber ?? 1
            }
            .interactiveDismissDisabled(busy)
        }
    }

    private func save() async {
        guard valid, !busy else { return }
        busy = true
        defer { busy = false }
        do {
            var body: [String: Any] = ["body": draft.body.trimmingCharacters(in: .whitespacesAndNewlines)]
            if let id = draft.noteID {
                try await session.mutate("me/notes/\(id)", method: "PATCH", body: body)
            } else {
                body["show_id"] = showID
                body["scope"] = draft.scope
                if draft.scope != "show" { body["season_number"] = draft.seasonNumber }
                if draft.scope == "episode" { body["episode_number"] = draft.episodeNumber }
                try await session.mutate("me/notes", body: body)
            }
            onSave()
            dismiss()
        } catch { self.error = error.localizedDescription }
    }
}
