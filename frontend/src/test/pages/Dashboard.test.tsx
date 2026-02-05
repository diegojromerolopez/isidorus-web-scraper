import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, vi, beforeEach, type MockedFunction } from 'vitest';
import axios from 'axios';
import Dashboard from '../../pages/Dashboard';

vi.mock('axios');
const mockedAxios = vi.mocked(axios, true);

vi.mock('../../assets/logo.png', () => ({
    default: 'mock-logo'
}));

describe('Dashboard', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.setItem('api_key', 'test-key');
    });

    it('renders clickable URLs in the scraping list', async () => {
        const mockScrapings = [
            { id: 123, url: 'http://example.com', user_id: 1, links_count: 5 },
        ];

        (mockedAxios.get as MockedFunction<typeof axios.get>).mockResolvedValueOnce({ data: { scrapings: mockScrapings } });

        render(
            <MemoryRouter>
                <Dashboard />
            </MemoryRouter>
        );

        // Wait for the data to load
        await waitFor(() => {
            expect(screen.getByText('http://example.com')).toBeInTheDocument();
        });

        const link = screen.getByRole('link', { name: 'http://example.com' });
        expect(link).toHaveAttribute('href', '/scraping/123');
    });
});
