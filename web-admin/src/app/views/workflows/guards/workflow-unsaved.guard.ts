import { CanDeactivateFn } from '@angular/router';
import { Observable } from 'rxjs';

export interface ComponentWithUnsavedChanges {
  canDeactivate: () => boolean | Observable<boolean> | Promise<boolean>;
}

export const workflowUnsavedGuard: CanDeactivateFn<ComponentWithUnsavedChanges> = (component) => {
  return component.canDeactivate ? component.canDeactivate() : true;
};
