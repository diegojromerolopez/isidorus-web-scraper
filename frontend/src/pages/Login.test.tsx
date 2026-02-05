import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { describe, it, expect, vi } from 'vitest';
import type { Mocked } from 'vitest';
import Login from './Login';
import axios from 'axios';

vi.mock('axios');
const mockedAxios = axios as Mocked<typeof axios>;

const renderLogin = () => {
    return render(
        <BrowserRouter>
            <Login />
        </BrowserRouter>
    );
};

describe('Login Component', () => {
    it('renders login form', () => {
        renderLogin();
        expect(screen.getByLabelText(/Username/i)).toBeDefined();
        expect(screen.getByLabelText(/Password/i)).toBeDefined();
        expect(screen.getByRole('button', { name: /Sign In/i })).toBeDefined();
    });

    it('handles successful login', async () => {
        mockedAxios.post.mockResolvedValueOnce({ data: { api_key: 'test-api-key' } });
        renderLogin();

        fireEvent.change(screen.getByLabelText(/Username/i), { target: { value: 'testuser' } });
        fireEvent.change(screen.getByLabelText(/Password/i), { target: { value: 'password123' } });
        fireEvent.click(screen.getByRole('button', { name: /Sign In/i }));

        await waitFor(() => {
            expect(mockedAxios.post).toHaveBeenCalledWith('http://localhost:8001/api/login/', {
                username: 'testuser',
                password: 'password123',
            });
            expect(localStorage.getItem('api_key')).toBe('test-api-key');
        });
    });

    it('displays error message on failed login', async () => {
        mockedAxios.post.mockRejectedValueOnce(new Error('Invalid credentials'));
        renderLogin();

        fireEvent.change(screen.getByLabelText(/Username/i), { target: { value: 'wronguser' } });
        fireEvent.change(screen.getByLabelText(/Password/i), { target: { value: 'wrongpass' } });
        fireEvent.click(screen.getByRole('button', { name: /Sign In/i }));

        await waitFor(() => {
            expect(screen.getByText(/Invalid credentials/i)).toBeDefined();
        });
    });
});
