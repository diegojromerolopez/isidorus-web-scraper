import React, { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import axios from 'axios';
import { Plus, Search, Loader2, LogOut, Trash2 } from 'lucide-react';
import logo from '../assets/logo.png';

const API_Base = 'http://localhost:8000'; // Direct to FastAPI

interface Scraping {
    id: number;
    url: string;
    user_id: number;
    links_count?: number;
}

export default function Dashboard() {
    const [scrapings, setScrapings] = useState<Scraping[]>([]);
    const [loading, setLoading] = useState(true);
    const [url, setUrl] = useState('');
    const [depth, setDepth] = useState(1);
    const [creating, setCreating] = useState(false);
    const navigate = useNavigate();

    useEffect(() => {
        fetchScrapings();
    }, []);

    const handleLogout = () => {
        localStorage.removeItem('api_key');
        navigate('/login');
    };

    const fetchScrapings = async () => {
        try {
            const apiKey = localStorage.getItem('api_key');
            const response = await axios.get(`${API_Base}/scrapings`, {
                headers: { 'X-API-Key': apiKey },
            });
            setScrapings(response.data.scrapings);
        } catch (error) {
            console.error('Failed to fetch scrapings', error);
        } finally {
            setLoading(false);
        }
    };

    const handleCreate = async (e: React.FormEvent) => {
        e.preventDefault();
        setCreating(true);
        try {
            const apiKey = localStorage.getItem('api_key');
            await axios.post(
                `${API_Base}/scrape`,
                { url, depth },
                { headers: { 'X-API-Key': apiKey } }
            );
            setUrl('');
            setDepth(1);
            fetchScrapings();
        } catch (error) {
            console.error('Failed to create scraping', error);
        } finally {
            setCreating(false);
        }
    };

    const handleDelete = async (id: number) => {
        if (!window.confirm('Are you sure you want to delete this scraping job? This action will remove all related data from the database, DynamoDB, and S3.')) {
            return;
        }

        try {
            const apiKey = localStorage.getItem('api_key');
            // Optimistic update
            setScrapings(prev => prev.filter(s => s.id !== id));

            await axios.delete(`${API_Base}/scraping/${id}`, {
                headers: { 'X-API-Key': apiKey }
            });

            // Delayed refresh to let the worker finish
            setTimeout(fetchScrapings, 2000);
        } catch (error) {
            console.error('Failed to delete scraping', error);
            alert('Failed to delete scraping job.');
            fetchScrapings(); // Revert on failure
        }
    };

    return (
        <div className="min-h-screen bg-slate-900 text-slate-100 selection:bg-sky-500/30">
            {/* Header / Nav */}
            <nav className="border-b border-slate-800 bg-slate-900/50 backdrop-blur-md sticky top-0 z-50">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                        <div className="h-9 w-9 bg-sky-500/10 rounded-lg flex items-center justify-center border border-sky-500/20 overflow-hidden">
                            <img src={logo} alt="Isidorus" className="w-full h-full object-cover" />
                        </div>
                        <span className="text-xl font-bold tracking-tight bg-gradient-to-r from-sky-400 to-blue-500 bg-clip-text text-transparent">
                            Isidorus
                        </span>
                    </div>
                    <button
                        onClick={handleLogout}
                        className="flex items-center gap-2 px-4 py-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-all duration-200"
                    >
                        <LogOut size={18} />
                        <span className="text-sm font-medium">Sign Out</span>
                    </button>
                </div>
            </nav>

            <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
                <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 mb-12">
                    <div className="space-y-1">
                        <h1 className="text-4xl font-extrabold tracking-tight text-white">Dashboard</h1>
                        <p className="text-slate-400">Monitor and launch your web scraping jobs</p>
                    </div>

                    <form onSubmit={handleCreate} className="flex flex-col md:flex-row gap-6 items-end bg-slate-800/40 p-6 rounded-3xl border border-slate-700/50 shadow-2xl backdrop-blur-xl">
                        <div className="flex-grow space-y-2 w-full">
                            <label htmlFor="url" className="block text-sm font-semibold text-slate-400 ml-1 uppercase tracking-wider">Target URL</label>
                            <div className="relative group">
                                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-slate-500 group-focus-within:text-sky-400 transition-colors" />
                                <input
                                    id="url"
                                    type="url"
                                    placeholder="https://example.com"
                                    required
                                    value={url}
                                    onChange={(e) => setUrl(e.target.value)}
                                    className="pl-10 pr-4 py-3 bg-slate-900/60 border border-slate-700/80 rounded-2xl text-white placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-sky-500/40 w-full transition-all hover:bg-slate-900"
                                />
                            </div>
                        </div>
                        <div className="space-y-2 w-full md:w-32">
                            <label htmlFor="depth" className="block text-sm font-semibold text-slate-400 ml-1 uppercase tracking-wider">Depth</label>
                            <input
                                id="depth"
                                type="number"
                                placeholder="1"
                                required
                                min="1"
                                max="5"
                                value={depth}
                                onChange={(e) => setDepth(parseInt(e.target.value))}
                                className="px-4 py-3 bg-slate-900/60 border border-slate-700/80 rounded-2xl text-white placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-sky-500/40 w-full transition-all hover:bg-slate-900"
                            />
                        </div>
                        <button
                            type="submit"
                            disabled={creating}
                            className="bg-sky-500 hover:bg-sky-400 text-slate-950 font-bold px-8 py-3 rounded-2xl flex items-center justify-center gap-2 transition-all active:scale-95 disabled:opacity-50 shadow-lg shadow-sky-500/20 w-full md:w-auto mt-2 md:mt-0 cursor-pointer h-[50px]"
                        >
                            {creating ? <Loader2 className="animate-spin h-5 w-5" /> : <Plus size={20} />}
                            <span className="uppercase tracking-tight">Launch</span>
                        </button>
                    </form>
                </div>

                <div className="grid grid-cols-1 gap-6">
                    {loading ? (
                        <div className="flex flex-col items-center justify-center py-20 bg-slate-800/30 rounded-3xl border border-slate-800 dashed">
                            <Loader2 className="animate-spin h-10 w-10 text-sky-500 mb-4" />
                            <p className="text-slate-400 animate-pulse">Retrieving your scraping history...</p>
                        </div>
                    ) : (
                        <div className="bg-slate-800/50 rounded-3xl border border-slate-700/50 overflow-hidden shadow-2xl">
                            <div className="overflow-x-auto">
                                <table className="w-full text-left border-collapse">
                                    <thead>
                                        <tr className="bg-slate-900/50">
                                            <th className="px-6 py-4 text-xs font-semibold text-slate-400 uppercase tracking-wider">Scraping URL</th>
                                            <th className="px-6 py-4 text-xs font-semibold text-slate-400 uppercase tracking-wider text-center">Links Found</th>
                                            <th className="px-6 py-4 text-xs font-semibold text-slate-400 uppercase tracking-wider">Internal ID</th>
                                            <th className="px-6 py-4 text-right"></th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-slate-700/50">
                                        {scrapings.map((scraping) => (
                                            <tr key={scraping.id} className="hover:bg-slate-700/30 transition-colors group">
                                                <td className="px-6 py-5">
                                                    <div className="flex items-center gap-3">
                                                        <div className="h-10 w-10 bg-sky-500/10 rounded-xl flex items-center justify-center group-hover:scale-110 transition-transform duration-200">
                                                            <Search className="h-5 w-5 text-sky-400" />
                                                        </div>
                                                        <Link
                                                            to={`/scraping/${scraping.id}`}
                                                            className="font-medium text-slate-200 break-all hover:text-sky-400 transition-colors"
                                                        >
                                                            {scraping.url}
                                                        </Link>
                                                    </div>
                                                </td>
                                                <td className="px-6 py-5 text-center">
                                                    <span className="px-3 py-1 bg-sky-500/10 text-sky-400 rounded-full text-sm font-bold border border-sky-500/20">
                                                        {scraping.links_count || 0}
                                                    </span>
                                                </td>
                                                <td className="px-6 py-5">
                                                    <span className="px-3 py-1 bg-slate-900 text-slate-400 rounded-md text-sm font-mono border border-slate-700">
                                                        #{scraping.id}
                                                    </span>
                                                </td>
                                                <td className="px-6 py-5 text-right">
                                                    <div className="flex items-center justify-end gap-2">
                                                        <Link
                                                            to={`/scraping/${scraping.id}`}
                                                            className="inline-flex items-center text-sm font-bold text-sky-400 hover:text-sky-300 transition-colors group/link"
                                                        >
                                                            Details
                                                            <Plus className="ml-1 h-4 w-4 transform group-hover/link:rotate-90 transition-transform" />
                                                        </Link>
                                                        <button
                                                            onClick={() => handleDelete(scraping.id)}
                                                            className="p-2 text-slate-500 hover:text-red-400 hover:bg-red-400/10 rounded-lg transition-all active:scale-90 cursor-pointer"
                                                            title="Delete Scraping"
                                                        >
                                                            <Trash2 size={18} />
                                                        </button>
                                                    </div>
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                            {scrapings.length === 0 && (
                                <div className="text-center py-20 px-4">
                                    <div className="h-20 w-20 bg-slate-700/30 rounded-full flex items-center justify-center mx-auto mb-6">
                                        <Search className="h-10 w-10 text-slate-600" />
                                    </div>
                                    <h3 className="text-xl font-bold text-white mb-2">No jobs found</h3>
                                    <p className="text-slate-400 max-w-sm mx-auto">
                                        Enter a URL above to start your first web scraping mission.
                                    </p>
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </main>
        </div>
    );
}
