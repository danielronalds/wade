// NOTE: Vibecoded and not suppppppper reviewed
import { onBeforeUnmount, onMounted } from 'vue';
import { registerEventHandlers } from '@/views/workspace/tabs/review/events/registerEventHandlers';

type WorkspaceEventHandlersOptions = {
  cancelReview: () => Promise<void>;
  getWorkspaceId: () => string;
  startReview: () => Promise<void>;
};

export const useWorkspaceEventHandlers = ({
  cancelReview,
  getWorkspaceId,
  startReview
}: WorkspaceEventHandlersOptions) => {
  let unregisterEventHandlers: (() => void) | undefined;

  onMounted(() => {
    unregisterEventHandlers = registerEventHandlers({
      cancelReview: async ({ workspaceId }) => {
        if (workspaceId !== getWorkspaceId()) {
          return;
        }

        await cancelReview();
      },
      startReview: async ({ workspaceId }) => {
        if (workspaceId !== getWorkspaceId()) {
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
