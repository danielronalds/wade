// NOTE: Vibecoded and not suppppppper reviewed
import { registerStartReviewEventHandler, type StartReviewEventHandler } from './startReview';

type RegisteredEventHandlers = {
  startReview?: StartReviewEventHandler;
};

export const registerEventHandlers = (handlers: RegisteredEventHandlers) => {
  // Event handlers are optional so each screen can subscribe only to events it understands while still receiving one cleanup callback.
  const unregisterHandlers = [
    handlers.startReview ? registerStartReviewEventHandler(handlers.startReview) : undefined
  ].filter((unregister): unregister is () => void => Boolean(unregister));

  return () => {
    unregisterHandlers.forEach((unregister) => unregister());
  };
};
