import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import axios from 'axios';
import { KeyRound, Loader2 } from 'lucide-react';
import logo from '../assets/logo.png';

const API_Base = 'http://localhost:8001';

export default function Login() {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const navigate = useNavigate();

    const handleLogin = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        setError('');

        try {
            const response = await axios.post(`${API_Base}/api/login/`, {
                username,
                password,
            });

            localStorage.setItem('api_key', response.data.api_key);
            navigate('/');
        } catch (err: unknown) {
            const errorMessage = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Invalid credentials';
            setError(errorMessage);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-slate-900 px-4">
            <div className="w-full max-w-md p-8 bg-slate-800/50 backdrop-blur-xl rounded-2xl shadow-2xl border border-slate-700/50 ring-1 ring-white/10">
                <div className="text-center mb-10">
                    <div className="mx-auto h-20 w-20 bg-sky-500/10 rounded-2xl flex items-center justify-center mb-6 rotate-3 hover:rotate-0 transition-all duration-300 overflow-hidden border border-sky-500/20">
                        <img src={logo} alt="Isidorus Logo" className="w-full h-full object-cover" />
                    </div>
                    <h2 className="text-3xl font-bold tracking-tight text-white mb-2">
                        Welcome Back
                    </h2>
                    <p className="text-slate-400 text-sm">
                        Sign in to manage your scrapings
                    </p>
                </div>
                <form onSubmit={handleLogin} className="space-y-6">
                    <div className="space-y-4">
                        <div className="space-y-1">
                            <label htmlFor="username" className="text-xs font-semibold text-slate-400 uppercase tracking-wider ml-1">Username</label>
                            <input
                                id="username"
                                type="text"
                                required
                                className="w-full px-4 py-3 bg-slate-900/50 border border-slate-700 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-sky-500/50 focus:border-sky-500 transition-all duration-200"
                                placeholder="Enter your username"
                                value={username}
                                onChange={(e) => setUsername(e.target.value)}
                            />
                        </div>
                        <div className="space-y-1">
                            <label htmlFor="password" className="text-xs font-semibold text-slate-400 uppercase tracking-wider ml-1">Password</label>
                            <input
                                id="password"
                                type="password"
                                required
                                className="w-full px-4 py-3 bg-slate-900/50 border border-slate-700 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-sky-500/50 focus:border-sky-500 transition-all duration-200"
                                placeholder="••••••••"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                            />
                        </div>
                    </div>

                    {error && (
                        <div className="bg-red-500/10 border border-red-500/20 text-red-400 p-3 rounded-lg text-sm text-center font-medium animate-pulse">
                            {error}
                        </div>
                    )}

                    <button
                        type="submit"
                        disabled={loading}
                        className="w-full flex justify-center items-center py-3.5 px-4 rounded-xl text-sm font-bold text-white bg-gradient-to-r from-sky-500 to-blue-600 hover:from-sky-400 hover:to-blue-500 active:scale-[0.98] transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed shadow-xl shadow-sky-500/25 group"
                    >
                        {loading ? (
                            <Loader2 className="animate-spin h-5 w-5" />
                        ) : (
                            <>
                                <span>Sign In</span>
                                <KeyRound className="ml-2 h-4 w-4 opacity-0 group-hover:opacity-100 -translate-x-2 group-hover:translate-x-0 transition-all duration-300" />
                            </>
                        )}
                    </button>
                </form>

                <div className="mt-8 pt-6 border-t border-slate-700/50 text-center">
                    <p className="text-xs text-slate-500 italic">
                        "Gathering and systematizing universal knowledge"
                    </p>
                </div>
            </div>
        </div>
    );
}
