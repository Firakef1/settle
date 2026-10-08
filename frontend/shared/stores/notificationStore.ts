import { create } from 'zustand';

export type ToastTone = 'success' | 'error' | 'info';

export interface Toast {
  id: string;
  message: string;
  tone: ToastTone;
}

interface NotificationState {
  toasts: Toast[];
  push: (message: string, tone?: ToastTone) => void;
  dismiss: (id: string) => void;
}

// App-wide toast queue. Rendered by <Toaster /> in the app shell.
export const useNotificationStore = create<NotificationState>((set, get) => ({
  toasts: [],
  push: (message, tone = 'info') => {
    const id = Math.random().toString(36).slice(2);
    set({ toasts: [...get().toasts, { id, message, tone }].slice(-4) });
    setTimeout(() => get().dismiss(id), 4500);
  },
  dismiss: (id) => set({ toasts: get().toasts.filter((t) => t.id !== id) }),
}));

export const toast = {
  success: (message: string) => useNotificationStore.getState().push(message, 'success'),
  error: (message: string) => useNotificationStore.getState().push(message, 'error'),
  info: (message: string) => useNotificationStore.getState().push(message, 'info'),
};
