// NOTE: Vibecoded and not suppppppper reviewed
import { onBeforeUnmount, onMounted } from 'vue';
import { registerEventHandlers } from '../events/registerEventHandlers';

type ProjectEventHandlersOptions = {
  cancelReview: () => Promise<void>;
  getProjectName: () => string;
  startReview: () => Promise<void>;
};

export const useProjectEventHandlers = ({
  cancelReview,
  getProjectName,
  startReview
}: ProjectEventHandlersOptions) => {
  let unregisterEventHandlers: (() => void) | undefined;

  onMounted(() => {
    unregisterEventHandlers = registerEventHandlers({
      cancelReview: async ({ projectName }) => {
        if (projectName !== getProjectName()) {
          return;
        }

        await cancelReview();
      },
      startReview: async ({ projectName }) => {
        if (projectName !== getProjectName()) {
          return;
        }

        await startReview();
      }
    });
  });

  onBeforeUnmount(() => {
    unregisterEventHandlers?.();
  });
};
