import UIKit
import UserNotifications

extension Notification.Name {
    /// Posted with the hex device token (String) once APNs registration succeeds.
    static let apnsDeviceTokenRegistered = Notification.Name("apnsDeviceTokenRegistered")
    /// Posted with a show id (Int) when the user taps a push about that show.
    static let pushOpenShow = Notification.Name("pushOpenShow")
}

/// Bridges UIKit's push-notification callbacks into the SwiftUI app via
/// NotificationCenter, so Session/MainTabsView don't need a reference to the
/// app delegate itself.
final class NotificationDelegate: NSObject, UIApplicationDelegate, UNUserNotificationCenterDelegate {
    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil) -> Bool {
        UNUserNotificationCenter.current().delegate = self
        return true
    }

    func application(_ application: UIApplication, didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
        let hex = deviceToken.map { String(format: "%02x", $0) }.joined()
        NotificationCenter.default.post(name: .apnsDeviceTokenRegistered, object: hex)
    }

    func application(_ application: UIApplication, didFailToRegisterForRemoteNotificationsWithError error: Error) {
        #if DEBUG
        print("APNs registration failed: \(error)")
        #endif
    }

    func userNotificationCenter(_ center: UNUserNotificationCenter, willPresent notification: UNNotification, withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void) {
        completionHandler([.banner, .sound, .list])
    }

    func userNotificationCenter(_ center: UNUserNotificationCenter, didReceive response: UNNotificationResponse, withCompletionHandler completionHandler: @escaping () -> Void) {
        if let showID = response.notification.request.content.userInfo["show_id"] as? String, let id = Int(showID) {
            NotificationCenter.default.post(name: .pushOpenShow, object: id)
        }
        completionHandler()
    }
}
