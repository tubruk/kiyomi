import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ProviderList } from './ProviderList';
import { ProviderRef, Source } from '../types/api';

const mockSources: Source[] = [
  {
    id: 'mangadex',
    name: 'MangaDex',
    icon: 'https://mangadex.org/favicon.ico',
    capabilities: ['content', 'metadata'],
  },
  {
    id: 'anilist',
    name: 'AniList',
    capabilities: ['metadata'],
  },
  {
    id: 'custom-source',
    name: 'Custom Source',
    capabilities: ['content'],
  },
];

const mockProviders: ProviderRef[] = [
  {
    provider_id: 'mangadex',
    provider_manga_id: 'md-123',
    manga_title: 'Frieren at the Funeral',
  },
  {
    provider_id: 'anilist',
    provider_manga_id: 'al-456',
    manga_title: 'Sousou no Frieren',
  },
  {
    provider_id: 'unknown-provider',
    provider_manga_id: 'unk-789',
  },
];

describe('ProviderList', () => {
  it('renders empty state message when providers array is empty', () => {
    render(
      <ProviderList
        providers={[]}
        sources={mockSources}
      />
    );

    expect(
      screen.getByText('No providers linked. Add from provider to enable sync + downloads.')
    ).toBeInTheDocument();
  });

  it('renders provider list items with names, icons, titles, and capability badges', () => {
    render(
      <ProviderList
        providers={mockProviders}
        sources={mockSources}
        contentProviderId="mangadex"
        contentProviderMangaId="md-123"
      />
    );

    // Provider names
    expect(screen.getByText('MangaDex')).toBeInTheDocument();
    expect(screen.getByText('AniList')).toBeInTheDocument();
    // Fallback to provider_id for unknown source
    expect(screen.getByText('unknown-provider')).toBeInTheDocument();

    // Manga titles
    expect(screen.getByText('Frieren at the Funeral')).toBeInTheDocument();
    expect(screen.getByText('Sousou no Frieren')).toBeInTheDocument();

    // Source icon
    const img = document.querySelector('img[src="https://mangadex.org/favicon.ico"]');
    expect(img).toBeInTheDocument();

    // Capability badges
    expect(screen.getByText('Content')).toBeInTheDocument();
    expect(screen.getAllByText('Metadata').length).toBeGreaterThanOrEqual(2);
  });

  it('renders active and unavailable capability state correctly', () => {
    render(
      <ProviderList
        providers={[mockProviders[0]]}
        sources={mockSources}
        contentProviderId="mangadex"
        contentProviderMangaId="md-123"
        isContentUnavailable={true}
      />
    );

    const contentBadge = screen.getByText('Content').closest('.group\\/badge, span');
    expect(contentBadge).toHaveAttribute('title', 'Content not available');
    expect(contentBadge).toHaveClass('border-amber-500/40');
  });

  it('triggers onImportMetadata when "Import metadata" menu item is clicked', async () => {
    const user = userEvent.setup();
    const handleImportMetadata = vi.fn();

    render(
      <ProviderList
        providers={[mockProviders[0]]}
        sources={mockSources}
        onImportMetadata={handleImportMetadata}
      />
    );

    const menuTrigger = screen.getByRole('button', { name: '•••' });
    await user.click(menuTrigger);

    const importItem = await screen.findByRole('menuitem', { name: /import metadata/i });
    expect(importItem).toBeInTheDocument();

    await user.click(importItem);
    expect(handleImportMetadata).toHaveBeenCalledTimes(1);
    expect(handleImportMetadata).toHaveBeenCalledWith(mockProviders[0]);
  });

  it('triggers onSwitchTo when "Switch content to this" menu item is clicked', async () => {
    const user = userEvent.setup();
    const handleSwitchTo = vi.fn();

    // mangadex is not active content provider here
    render(
      <ProviderList
        providers={[mockProviders[0]]}
        sources={mockSources}
        contentProviderId="other-provider"
        contentProviderMangaId="other-id"
        onSwitchTo={handleSwitchTo}
      />
    );

    const menuTrigger = screen.getByRole('button', { name: '•••' });
    await user.click(menuTrigger);

    const switchItem = await screen.findByRole('menuitem', { name: /switch content to this/i });
    expect(switchItem).toBeInTheDocument();

    await user.click(switchItem);
    expect(handleSwitchTo).toHaveBeenCalledTimes(1);
    expect(handleSwitchTo).toHaveBeenCalledWith(mockProviders[0]);
  });

  it('does not display "Switch content to this" when provider is already active content provider', async () => {
    const user = userEvent.setup();

    render(
      <ProviderList
        providers={[mockProviders[0]]}
        sources={mockSources}
        contentProviderId="mangadex"
        contentProviderMangaId="md-123"
      />
    );

    const menuTrigger = screen.getByRole('button', { name: '•••' });
    await user.click(menuTrigger);

    expect(screen.queryByRole('menuitem', { name: /switch content to this/i })).not.toBeInTheDocument();
  });

  it('does not display "Switch content to this" when provider lacks content capability', async () => {
    const user = userEvent.setup();

    // anilist only has 'metadata' capability
    render(
      <ProviderList
        providers={[mockProviders[1]]}
        sources={mockSources}
      />
    );

    const menuTrigger = screen.getByRole('button', { name: '•••' });
    await user.click(menuTrigger);

    expect(screen.queryByRole('menuitem', { name: /switch content to this/i })).not.toBeInTheDocument();
  });

  it('triggers onRemove when "Remove" menu item is clicked', async () => {
    const user = userEvent.setup();
    const handleRemove = vi.fn();

    render(
      <ProviderList
        providers={[mockProviders[0]]}
        sources={mockSources}
        onRemove={handleRemove}
      />
    );

    const menuTrigger = screen.getByRole('button', { name: '•••' });
    await user.click(menuTrigger);

    const removeItem = await screen.findByRole('menuitem', { name: /remove/i });
    expect(removeItem).toBeInTheDocument();

    await user.click(removeItem);
    expect(handleRemove).toHaveBeenCalledTimes(1);
    expect(handleRemove).toHaveBeenCalledWith(mockProviders[0]);
  });

  it('disables "Remove" menu item when canRemoveProvider returns false', async () => {
    const user = userEvent.setup();
    const handleRemove = vi.fn();

    render(
      <ProviderList
        providers={[mockProviders[0]]}
        sources={mockSources}
        onRemove={handleRemove}
        canRemoveProvider={() => false}
      />
    );

    const menuTrigger = screen.getByRole('button', { name: '•••' });
    await user.click(menuTrigger);

    const removeItem = await screen.findByRole('menuitem', { name: /remove/i });
    expect(removeItem).toHaveAttribute('aria-disabled', 'true');

    await user.click(removeItem);
    expect(handleRemove).not.toHaveBeenCalled();
  });

  it('disables "Remove" menu item when isRemoving is true', async () => {
    const user = userEvent.setup();
    const handleRemove = vi.fn();

    render(
      <ProviderList
        providers={[mockProviders[0]]}
        sources={mockSources}
        onRemove={handleRemove}
        isRemoving={true}
      />
    );

    const menuTrigger = screen.getByRole('button', { name: '•••' });
    await user.click(menuTrigger);

    const removeItem = await screen.findByRole('menuitem', { name: /remove/i });
    expect(removeItem).toHaveAttribute('aria-disabled', 'true');

    await user.click(removeItem);
    expect(handleRemove).not.toHaveBeenCalled();
  });
});
