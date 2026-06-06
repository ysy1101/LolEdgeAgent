import React from 'react';

interface Props { children: React.ReactNode; fallback?: React.ReactNode; }
interface State { hasError: boolean; error?: Error; }

export class ErrorBoundary extends React.Component<Props, State> {
  state: State = { hasError: false };
  static getDerivedStateFromError(error: Error) { return { hasError: true, error }; }
  render() {
    if (this.state.hasError) {
      return this.props.fallback || (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          渲染出错，请刷新页面重试。
          <details className="mt-1 text-xs text-red-500">{this.state.error?.message}</details>
        </div>
      );
    }
    return this.props.children;
  }
}
