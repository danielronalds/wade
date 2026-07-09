const startReviewEventName = 'wade:start-review';

export type StartReviewEventDetail = {
  projectName: string;
};

export type StartReviewEventHandler = (detail: StartReviewEventDetail) => void | Promise<void>;

export const dispatchStartReviewEvent = (projectName: string) => {
  window.dispatchEvent(new CustomEvent<StartReviewEventDetail>(startReviewEventName, {
    detail: { projectName }
  }));
};

export const registerStartReviewEventHandler = (handler: StartReviewEventHandler) => {
  const listener = (event: Event) => {
    void handler((event as CustomEvent<StartReviewEventDetail>).detail);
  };

  window.addEventListener(startReviewEventName, listener);

  return () => {
    window.removeEventListener(startReviewEventName, listener);
  };
};
