import { Routes } from '@angular/router';
import { workflowUnsavedGuard } from './guards/workflow-unsaved.guard';

export const routes: Routes = [
  {
    path: '',
    data: {
      title: 'Workflows'
    },
    children: [
      {
        path: '',
        loadComponent: () => import('./workflows.component').then(m => m.WorkflowsComponent),
        data: {
          title: ''
        }
      },
      {
        path: 'new',
        loadComponent: () => import('./workflow-editor/workflow-editor.component').then(m => m.WorkflowEditorComponent),
        canDeactivate: [workflowUnsavedGuard],
        data: {
          title: 'Create Workflow'
        }
      },
      {
        path: ':id/edit',
        loadComponent: () => import('./workflow-editor/workflow-editor.component').then(m => m.WorkflowEditorComponent),
        canDeactivate: [workflowUnsavedGuard],
        data: {
          title: 'Edit Workflow'
        }
      },
      {
        path: ':workflowId/executions',
        loadComponent: () => import('./workflow-executions/workflow-executions.component').then(m => m.WorkflowExecutionsComponent),
        data: {
          title: 'Executions'
        }
      },
      {
        path: ':workflowId/executions/:executionId',
        loadComponent: () => import('./workflow-executions/workflow-executions.component').then(m => m.WorkflowExecutionsComponent),
        data: {
          title: 'Execution Detail'
        }
      },
      {
        path: ':workflowId/versions',
        loadComponent: () => import('./workflow-version-history/workflow-version-history.component').then(m => m.WorkflowVersionHistoryComponent),
        data: {
          title: 'Version History'
        }
      }
    ]
  }
];
