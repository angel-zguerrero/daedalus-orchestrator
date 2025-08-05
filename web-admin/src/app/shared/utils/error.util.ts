import { HttpErrorResponse } from '@angular/common/http';

export interface FormattedError {
  message: string;
  code?: string | number;
  details?: string;
}

export class ErrorUtil {
  /**
   * Formats an error response to display both error code and server message
   * @param error - The error object from HTTP response
   * @returns Formatted error object with message, code and details
   */
  static formatError(error: any): FormattedError {
    let message = 'An unexpected error occurred';
    let code: string | number | undefined = undefined;
    let details: string | undefined = undefined;

    if (error instanceof HttpErrorResponse) {
      // HTTP error response
      code = error.status;
      
      if (error.error) {
        if (typeof error.error === 'string') {
          // Simple string error
          message = error.error;
        } else if (error.error.message) {
          // Error object with message
          message = error.error.message;
          if (error.error.code) {
            code = error.error.code;
          }
          if (error.error.details) {
            details = error.error.details;
          }
        } else if (error.error.error) {
          // Nested error structure
          message = error.error.error;
        }
      } else if (error.message) {
        message = error.message;
      }
      
      // Add status text if available and different from message
      if (error.statusText && error.statusText !== 'Unknown Error' && !message.includes(error.statusText)) {
        details = error.statusText;
      }
    } else if (error.error) {
      // Non-HTTP error with error property
      if (typeof error.error === 'string') {
        message = error.error;
      } else if (error.error.message) {
        message = error.error.message;
        if (error.error.code) {
          code = error.error.code;
        }
      }
    } else if (error.message) {
      // Error with message property
      message = error.message;
      if (error.code) {
        code = error.code;
      }
    } else if (typeof error === 'string') {
      // String error
      message = error;
    }

    return { message, code, details };
  }

  /**
   * Formats error for display in UI components
   * @param error - The error object from HTTP response
   * @returns HTML string with formatted error message
   */
  static formatErrorMessage(error: any): string {
    const formattedError = this.formatError(error);
    
    let errorMessage = formattedError.message;
    
    if (formattedError.code) {
      errorMessage = `[${formattedError.code}] ${errorMessage}`;
    }
    
    if (formattedError.details) {
      errorMessage += ` (${formattedError.details})`;
    }
    
    return errorMessage;
  }
}
