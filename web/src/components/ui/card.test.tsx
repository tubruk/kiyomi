import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from './card';

describe('UI/Card', () => {
  it('renders complete card structure', () => {
    render(
      <Card data-testid="card-root">
        <CardHeader>
          <CardTitle>Test Card Title</CardTitle>
          <CardDescription>Test Card Description</CardDescription>
        </CardHeader>
        <CardContent>
          <p>Card body content</p>
        </CardContent>
        <CardFooter>
          <span>Footer note</span>
        </CardFooter>
      </Card>
    );

    expect(screen.getByTestId('card-root')).toHaveClass('bg-card');
    expect(screen.getByText('Test Card Title')).toBeInTheDocument();
    expect(screen.getByText('Test Card Description')).toBeInTheDocument();
    expect(screen.getByText('Card body content')).toBeInTheDocument();
    expect(screen.getByText('Footer note')).toBeInTheDocument();
  });
});
