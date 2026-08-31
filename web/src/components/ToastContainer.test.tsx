import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ToastContainer } from './ToastContainer';
import { ToastProvider, useToast } from '../context/ToastContext';
import React from 'react';

const TestTrigger: React.FC<{
  onTrigger: (showToast: ReturnType<typeof useToast>['showToast']) => void;
}> = ({ onTrigger }) => {
  const { showToast } = useToast();
  return (
    <button onClick={() => onTrigger(showToast)}>
      Trigger Toast
    </button>
  );
};

describe('ToastContainer', () => {
  it('renders nothing when there are no toasts', () => {
    const { container } = render(
      <ToastProvider>
        <ToastContainer />
      </ToastProvider>
    );

    expect(container.firstChild).toBeNull();
  });

  it('renders success toast and allows dismissing', async () => {
    const user = userEvent.setup();
    render(
      <ToastProvider>
        <TestTrigger
          onTrigger={(showToast) => {
            showToast('Item saved successfully!', 'success');
          }}
        />
        <ToastContainer />
      </ToastProvider>
    );

    await user.click(screen.getByRole('button', { name: 'Trigger Toast' }));

    expect(screen.getByText('Item saved successfully!')).toBeInTheDocument();

    const dismissBtn = screen.getByRole('button', { name: 'Dismiss toast' });
    await user.click(dismissBtn);

    expect(screen.queryByText('Item saved successfully!')).not.toBeInTheDocument();
  });

  it('renders error toast with Details button that opens ErrorDetailsModal', async () => {
    const user = userEvent.setup();
    render(
      <ToastProvider>
        <TestTrigger
          onTrigger={(showToast) => {
            showToast('Download failed', 'error', 'Error code 500: Server error');
          }}
        />
        <ToastContainer />
      </ToastProvider>
    );

    await user.click(screen.getByRole('button', { name: 'Trigger Toast' }));

    expect(screen.getByText('Download failed')).toBeInTheDocument();

    const detailsBtn = screen.getByRole('button', { name: 'Details' });
    await user.click(detailsBtn);

    expect(screen.getByText('Error Details')).toBeInTheDocument();
    expect(screen.getByText('Error code 500: Server error')).toBeInTheDocument();
  });

  it('renders subtle mode toast', async () => {
    const user = userEvent.setup();
    render(
      <ToastProvider>
        <TestTrigger
          onTrigger={(showToast) => {
            showToast('Subtle notification', 'info', undefined, 'subtle');
          }}
        />
        <ToastContainer />
      </ToastProvider>
    );

    await user.click(screen.getByRole('button', { name: 'Trigger Toast' }));
    expect(screen.getByText('Subtle notification')).toBeInTheDocument();
  });
});
