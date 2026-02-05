import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { Mocked } from 'vitest';
import ScrapingDetail from './ScrapingDetail';
import axios from 'axios';

vi.mock('axios');
const mockedAxios = axios as Mocked<typeof axios>;

const renderDetail = () => {
    return render(
        <BrowserRouter>
            <Routes>
                <Route path="/scraping/:id" element={<ScrapingDetail />} />
            </Routes>
        </BrowserRouter>
    );
};

describe('ScrapingDetail Component', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.setItem('api_key', 'test-api-key');
        window.history.pushState({}, '', '/scraping/1');
    });

    it('shows pending status then completed data', async () => {
        // First call returns pending
        mockedAxios.get.mockResolvedValueOnce({
            data: { status: 'PENDING', scraping: { id: 1, url: 'https://test.com', depth: 1 } }
        });
        // Second call returns completed
        mockedAxios.get.mockResolvedValueOnce({
            data: {
                status: 'COMPLETED',
                scraping: { id: 1, url: 'https://test.com', depth: 1 },
                data: [{
                    url: 'https://test.com',
                    summary: 'Test summary',
                    terms: [{ term: 'test', frequency: 5 }],
                    images: [{ url: 'http://img.com/1.jpg', explanation: 'An image' }]
                }]
            }
        });

        renderDetail();

        await waitFor(() => {
            expect(mockedAxios.get).toHaveBeenCalledWith('http://localhost:8000/scrape?scraping_id=1', expect.any(Object));
            expect(screen.getByText(/PENDING/i)).toBeDefined();
        });

        // The interval is 3000ms. waitFor default timeout is 1000ms.
        // We wait for the second call to happen.
        await waitFor(() => {
            expect(screen.getByText(/Test summary/i)).toBeDefined();
            expect(screen.getByText(/An image/i)).toBeDefined();
        }, { timeout: 5000 });
    });

    it('handles rerun action', async () => {
        mockedAxios.get.mockResolvedValueOnce({
            data: { status: 'COMPLETED', scraping: { id: 1, url: 'https://test.com', depth: 2 } }
        });
        mockedAxios.post.mockResolvedValueOnce({ data: { scraping_id: 3 } });

        renderDetail();

        await waitFor(() => {
            expect(screen.getByTitle(/Re-run Scrape/i)).toBeDefined();
        });

        fireEvent.click(screen.getByTitle(/Re-run Scrape/i));

        await waitFor(() => {
            expect(mockedAxios.post).toHaveBeenCalledWith(
                'http://localhost:8000/scrape',
                { url: 'https://test.com', depth: 2 },
                { headers: { 'X-API-Key': 'test-api-key' } }
            );
        });
    });
});
