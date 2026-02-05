import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, it, expect, vi, beforeEach, type MockedFunction } from 'vitest';
import axios from 'axios';
import ScrapingDetail from '../../pages/ScrapingDetail';

vi.mock('axios');
const mockedAxios = vi.mocked(axios, true);

vi.mock('../../assets/logo.png', () => ({
    default: 'mock-logo'
}));

describe('ScrapingDetail', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.setItem('api_key', 'test-key');
    });

    it('toggles collapsible sections when clicking headers', async () => {
        const mockScraping = {
            id: 1,
            url: 'http://root.com',
            status: 'COMPLETED',
            pages: [
                {
                    url: 'http://child.com',
                    summary: 'This is a summary',
                    images: [],
                    terms: [],
                },
            ],
        };

        (mockedAxios.get as MockedFunction<typeof axios.get>).mockResolvedValueOnce({ data: { scraping: mockScraping } });

        render(
            <MemoryRouter initialEntries={['/scraping/1']}>
                <Routes>
                    <Route path="/scraping/:id" element={<ScrapingDetail />} />
                </Routes>
            </MemoryRouter>
        );

        // Wait for data to load
        await waitFor(() => {
            expect(screen.getByText('http://child.com')).toBeInTheDocument();
        });

        // Content should NOT be visible initially (collapsed)
        expect(screen.queryByText('AI Intelligence Summary')).not.toBeInTheDocument();

        // Click the toggle header
        const toggleButton = screen.getByRole('button', { name: /http:\/\/child\.com/i });
        fireEvent.click(toggleButton);

        // Content should now be visible
        expect(await screen.findByText('AI Intelligence Summary')).toBeInTheDocument();
        expect(screen.getByText('"This is a summary"')).toBeInTheDocument();

        // Click again to collapse
        fireEvent.click(toggleButton);
        expect(screen.queryByText('AI Intelligence Summary')).not.toBeInTheDocument();
    });
});
