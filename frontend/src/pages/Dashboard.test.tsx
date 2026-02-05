import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { Mocked } from 'vitest';
import Dashboard from './Dashboard';
import axios from 'axios';

vi.mock('axios');
const mockedAxios = axios as Mocked<typeof axios>;

const renderDashboard = () => {
    return render(
        <BrowserRouter>
            <Dashboard />
        </BrowserRouter>
    );
};

describe('Dashboard Component', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.setItem('api_key', 'test-api-key');
    });

    it('renders dashboard and fetches scrapings', async () => {
        mockedAxios.get.mockResolvedValueOnce({
            data: { data: [{ id: 1, url: 'https://example.com', user_id: 1 }] }
        });

        renderDashboard();

        expect(screen.getByText(/Retrieving your scraping history.../i)).toBeDefined();

        await waitFor(() => {
            expect(screen.getByText('https://example.com')).toBeDefined();
            expect(screen.getByText('#1')).toBeDefined();
        });
    });

    it('handles starting a new scrape', async () => {
        mockedAxios.get.mockResolvedValueOnce({ data: { data: [] } }); // Initial empty list
        mockedAxios.post.mockResolvedValueOnce({ data: { scraping_id: 2 } });
        mockedAxios.get.mockResolvedValueOnce({
            data: { data: [{ id: 2, url: 'https://newsite.com', user_id: 1 }] }
        }); // List after create

        renderDashboard();

        fireEvent.change(screen.getByPlaceholderText(/https:\/\/example\.com/i), { target: { value: 'https://newsite.com' } });
        fireEvent.change(screen.getByPlaceholderText(/Depth/i), { target: { value: '2' } });
        fireEvent.click(screen.getByRole('button', { name: /Launch Scrape/i }));

        await waitFor(() => {
            expect(mockedAxios.post).toHaveBeenCalledWith(
                'http://localhost:8000/scrape',
                { url: 'https://newsite.com', depth: 2 },
                { headers: { 'X-API-Key': 'test-api-key' } }
            );
            expect(screen.getByText('https://newsite.com')).toBeDefined();
        });
    });

    it('handles logout', async () => {
        mockedAxios.get.mockResolvedValueOnce({ data: { data: [] } });
        renderDashboard();

        fireEvent.click(screen.getByRole('button', { name: /Sign Out/i }));

        expect(localStorage.getItem('api_key')).toBeNull();
    });
});
