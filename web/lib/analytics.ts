/**
 * Type declarations and helper functions for Umami Analytics.
 */

declare global {
  interface Window {
    umami?: {
      track: {
        (eventName: string, eventData?: Record<string, string | number | boolean>): void;
        (customPayload: (defaultProps: Record<string, unknown>) => Record<string, unknown>): void;
      };
    };
  }
}

/**
 * Dispatches a custom event to Umami Analytics if running in the browser and tracker is initialized.
 *
 * @param eventName Identifier for the event (e.g. 'patient_registered', 'ticket_called', 'login_success')
 * @param data Key-value pairs for additional event context
 */
export function trackEvent(
  eventName: string,
  data?: Record<string, string | number | boolean>
): void {
  if (typeof window !== "undefined" && window.umami) {
    try {
      window.umami.track(eventName, data);
    } catch (err) {
      console.warn("[Umami Analytics] Failed to track event:", err);
    }
  }
}
