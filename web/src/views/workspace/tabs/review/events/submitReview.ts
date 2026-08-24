const submitReviewEventName = 'wade:submit-review';

export type SubmitReviewEventDetail = {
  workspaceId: string;
};

export type SubmitReviewEventHandler = (detail: SubmitReviewEventDetail) => void | Promise<void>;

export const dispatchSubmitReviewEvent = (workspaceId: string) => {
  window.dispatchEvent(
    new CustomEvent<SubmitReviewEventDetail>(submitReviewEventName, {
      detail: { workspaceId }
    })
  );
};

export const registerSubmitReviewEventHandler = (handler: SubmitReviewEventHandler) => {
  const listener = (event: Event) => {
    void handler((event as CustomEvent<SubmitReviewEventDetail>).detail);
  };

  window.addEventListener(submitReviewEventName, listener);

  return () => {
    window.removeEventListener(submitReviewEventName, listener);
  };
};
