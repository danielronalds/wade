// NOTE: Vibecoded and not suppppppper reviewed
import { registerCancelReviewEventHandler, type CancelReviewEventHandler } from './cancelReview';
import { registerStartReviewEventHandler, type StartReviewEventHandler } from './startReview';

type RegisteredEventHandlers = {
  cancelReview?: CancelReviewEventHandler;
  startReview?: StartReviewEventHandler;
};

export const registerEventHandlers = (handlers: RegisteredEventHandlers) => {
  // Event handlers are optional so each screen can subscribe only to events it understands while still receiving one cleanup callback.
  const unregisterHandlers = [
    handlers.cancelReview ? registerCancelReviewEventHandler(handlers.cancelReview) : undefined,
    handlers.startReview ? registerStartReviewEventHandler(handlers.startReview) : undefined
  ].filter((unregister): unregister is () => void => Boolean(unregister));

  return () => {
    unregisterHandlers.forEach((unregister) => unregister());
  };
};
