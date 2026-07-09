const cancelReviewEventName = 'wade:cancel-review';

export type CancelReviewEventDetail = {
  projectName: string;
};

export type CancelReviewEventHandler = (detail: CancelReviewEventDetail) => void | Promise<void>;

export const dispatchCancelReviewEvent = (projectName: string) => {
  window.dispatchEvent(new CustomEvent<CancelReviewEventDetail>(cancelReviewEventName, {
    detail: { projectName }
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
