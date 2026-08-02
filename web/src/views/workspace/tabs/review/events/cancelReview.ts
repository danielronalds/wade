const cancelReviewEventName = 'wade:cancel-review';

export type CancelReviewEventDetail = {
  workspaceId: string;
};

export type CancelReviewEventHandler = (detail: CancelReviewEventDetail) => void | Promise<void>;

export const dispatchCancelReviewEvent = (workspaceId: string) => {
  window.dispatchEvent(new CustomEvent<CancelReviewEventDetail>(cancelReviewEventName, {
    detail: { workspaceId }
  }));
};

export const registerCancelReviewEventHandler = (handler: CancelReviewEventHandler) => {
  const listener = (event: Event) => {
    void handler((event as CustomEvent<CancelReviewEventDetail>).detail);
  };

  window.addEventListener(cancelReviewEventName, listener);

  return () => {
    window.removeEventListener(cancelReviewEventName, listener);
  };
};
