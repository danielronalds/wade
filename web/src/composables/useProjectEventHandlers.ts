// NOTE: Vibecoded and not suppppppper reviewed
import { onBeforeUnmount, onMounted } from 'vue';
import { registerEventHandlers } from '../events/registerEventHandlers';

type ProjectEventHandlersOptions = {
  getProjectName: () => string;
  startReview: () => Promise<void>;
};

export const useProjectEventHandlers = ({
  getProjectName,
  startReview
}: ProjectEventHandlersOptions) => {
  let unregisterEventHandlers: (() => void) | undefined;

  onMounted(() => {
    unregisterEventHandlers = registerEventHandlers({
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
