import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, vi, beforeEach, type MockedFunction } from 'vitest';
import axios from 'axios';
import Login from '../../pages/Login';

vi.mock('axios');
const mockedAxios = vi.mocked(axios, true);

vi.mock('../../assets/logo.png', () => ({
    default: 'mock-logo'
}));

describe('Login Component', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();
    });

    it('renders login form', () => {
        render(
            <MemoryRouter>
                <Login />
            </MemoryRouter>
        );
        expect(screen.getByLabelText(/Username/i)).toBeInTheDocument();
        expect(screen.getByLabelText(/Password/i)).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /Sign In/i })).toBeInTheDocument();
    });

    it('handles successful login', async () => {
        (mockedAxios.post as MockedFunction<typeof axios.post>).mockResolvedValueOnce({ data: { api_key: 'test-api-key' } });

        render(
            <MemoryRouter>
                <Login />
            </MemoryRouter>
        );

        fireEvent.change(screen.getByLabelText(/Username/i), { target: { value: 'testuser' } });
        fireEvent.change(screen.getByLabelText(/Password/i), { target: { value: 'password123' } });
        fireEvent.click(screen.getByRole('button', { name: /Sign In/i }));

        await waitFor(() => {
            expect(mockedAxios.post).toHaveBeenCalled();
            expect(localStorage.getItem('api_key')).toBe('test-api-key');
        });
    });

    it('displays error message on failed login', async () => {
        (mockedAxios.post as MockedFunction<typeof axios.post>).mockRejectedValueOnce({
            response: { data: { detail: 'Invalid credentials' } }
        });

        render(
            <MemoryRouter>
                <Login />
            </MemoryRouter>
        );

        fireEvent.change(screen.getByLabelText(/Username/i), { target: { value: 'wronguser' } });
        fireEvent.change(screen.getByLabelText(/Password/i), { target: { value: 'wrongpass' } });
        fireEvent.click(screen.getByRole('button', { name: /Sign In/i }));

        await waitFor(() => {
            expect(screen.getByText(/Invalid credentials/i)).toBeInTheDocument();
        });
    });
});
