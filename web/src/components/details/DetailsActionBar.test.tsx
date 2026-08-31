import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DetailsActionBar } from './DetailsActionBar';

vi.mock('@tanstack/react-router', () => ({
  Link: ({ children, to, ...props }: any) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}));

describe('DetailsActionBar', () => {
  it('renders "Back to Explore" and "Add to Library" on remote routes', () => {
    const onAdd = vi.fn();
    render(
      <DetailsActionBar
        isRemoteRoute={true}
        providerIdParam="mangafox"
        isInLibrary={false}
        manga={{ id: 'm-1', title: 'Remote Manga' }}
        onAddToLibrary={onAdd}
      />
    );

    expect(screen.getByText('Back to Explore')).toBeInTheDocument();
    const addBtn = screen.getByRole('button', { name: /add to library/i });
    expect(addBtn).toBeInTheDocument();

    fireEvent.click(addBtn);
    expect(onAdd).toHaveBeenCalled();
  });

  it('renders "View in Library" link when remote manga is already in library', () => {
    render(
      <DetailsActionBar
        isRemoteRoute={true}
        providerIdParam="mangafox"
        isInLibrary={true}
        targetMangaId="local-1"
        manga={{ id: 'local-1', title: 'Saved Manga' }}
      />
    );

    expect(screen.getByText('View in Library')).toBeInTheDocument();
  });

  it('renders "Back to Library" and reading CTA button on local library view', () => {
    const onPrimaryCta = vi.fn();
    render(
      <DetailsActionBar
        isRemoteRoute={false}
        isInLibrary={true}
        targetMangaId="local-1"
        manga={{ id: 'local-1', title: 'Local Manga' }}
        hasChapters={true}
        readingCta={{ label: 'Continue Ch. 5', chapterId: 'ch-5', page: 1 }}
        onPrimaryCta={onPrimaryCta}
      />
    );

    expect(screen.getByText('Back to Library')).toBeInTheDocument();
    const ctaBtn = screen.getByRole('button', { name: /continue ch\. 5/i });
    expect(ctaBtn).toBeInTheDocument();

    fireEvent.click(ctaBtn);
    expect(onPrimaryCta).toHaveBeenCalled();
  });
});
