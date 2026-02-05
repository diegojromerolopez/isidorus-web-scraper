import React, { useEffect, useState } from 'react';
import { useLocation, Link, useNavigate } from 'react-router-dom';
import axios from 'axios';
import { Search as SearchIcon, Loader2, ArrowLeft, ExternalLink, Calendar, Layout } from 'lucide-react';
import logo from '../assets/logo.png';

const API_Base = 'http://localhost:8000';

interface SearchResult {
    url: string;
    scraping_id: number;
    created_at: string;
    highlights: string[];
}

export default function Search() {
    const location = useLocation();
    const navigate = useNavigate();
    const queryParams = new URLSearchParams(location.search);
    const query = queryParams.get('q') || '';

    const [results, setResults] = useState<SearchResult[]>([]);
    const [loading, setLoading] = useState(false);
    const [searchInput, setSearchInput] = useState(query);

    useEffect(() => {
        if (query) {
            performSearch(query);
        }
    }, [query]);

    const performSearch = async (searchTerm: string) => {
        setLoading(true);
        try {
            const apiKey = localStorage.getItem('api_key');
            const response = await axios.get(`${API_Base}/search?t=${encodeURIComponent(searchTerm)}`, {
                headers: { 'X-API-Key': apiKey },
            });
            setResults(response.data.results);
        } catch (error) {
            console.error('Search failed', error);
        } finally {
            setLoading(false);
        }
    };

    const handleSearch = (e: React.FormEvent) => {
        e.preventDefault();
        if (searchInput.trim()) {
            navigate(`/search?q=${encodeURIComponent(searchInput.trim())}`);
        }
    };

    return (
        <div className="min-h-screen bg-slate-900 text-slate-100 selection:bg-sky-500/30">
            {/* Nav */}
            <nav className="border-b border-slate-800 bg-slate-900/50 backdrop-blur-md sticky top-0 z-50">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                        <Link to="/" className="flex items-center gap-3 hover:opacity-80 transition-opacity">
                            <div className="h-9 w-9 bg-sky-500/10 rounded-lg flex items-center justify-center border border-sky-500/20 overflow-hidden">
                                <img src={logo} alt="Isidorus" className="w-full h-full object-cover" />
                            </div>
                            <span className="text-xl font-bold tracking-tight bg-gradient-to-r from-sky-400 to-blue-500 bg-clip-text text-transparent">
                                Isidorus
                            </span>
                        </Link>
                    </div>

                    <form onSubmit={handleSearch} className="hidden md:flex flex-grow max-w-md mx-8">
                        <div className="relative w-full group">
                            <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500 group-focus-within:text-sky-400" />
                            <input
                                type="text"
                                placeholder="Search everything..."
                                value={searchInput}
                                onChange={(e) => setSearchInput(e.target.value)}
                                className="w-full pl-10 pr-4 py-2 bg-slate-800/50 border border-slate-700/80 rounded-xl text-sm text-white focus:outline-none focus:ring-2 focus:ring-sky-500/40 transition-all"
                            />
                        </div>
                    </form>

                    <Link to="/" className="text-sm font-medium text-slate-400 hover:text-white transition-colors">
                        Dashboard
                    </Link>
                </div>
            </nav>

            <main className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
                <div className="mb-10">
                    <Link to="/" className="inline-flex items-center text-sm font-medium text-sky-400 hover:text-sky-300 mb-6 transition-colors">
                        <ArrowLeft size={16} className="mr-2" />
                        Back to Dashboard
                    </Link>
                    <h1 className="text-3xl font-extrabold text-white mb-2">
                        Search Results
                    </h1>
                    <p className="text-slate-400">
                        {loading ? 'Searching...' : `Found ${results.length} matches for "${query}"`}
                    </p>
                </div>

                {loading ? (
                    <div className="flex flex-col items-center justify-center py-20 bg-slate-800/20 rounded-3xl border border-slate-800/50 border-dashed">
                        <Loader2 className="animate-spin h-10 w-10 text-sky-500 mb-4" />
                        <p className="text-slate-400 animate-pulse font-medium">Scouring the web archives...</p>
                    </div>
                ) : (
                    <div className="space-y-6">
                        {results.map((result, idx) => (
                            <div key={idx} className="bg-slate-800/40 p-6 rounded-2xl border border-slate-700/50 hover:border-sky-500/30 transition-all group shadow-sm hover:shadow-sky-500/5">
                                <div className="flex flex-col md:flex-row md:items-start justify-between gap-4 mb-4">
                                    <div className="space-y-1">
                                        <a
                                            href={result.url}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="text-lg font-bold text-sky-400 hover:text-sky-300 flex items-center gap-2 group/title"
                                        >
                                            {result.url}
                                            <ExternalLink size={14} className="opacity-0 group-hover/title:opacity-100 transition-opacity" />
                                        </a>
                                        <div className="flex items-center gap-4 text-xs font-medium text-slate-500 uppercase tracking-widest">
                                            <span className="flex items-center gap-1">
                                                <Calendar size={12} />
                                                {new Date(result.created_at).toLocaleDateString()}
                                            </span>
                                            <Link to={`/scraping/${result.scraping_id}`} className="flex items-center gap-1 hover:text-sky-400 transition-colors">
                                                <Layout size={12} />
                                                Job #{result.scraping_id}
                                            </Link>
                                        </div>
                                    </div>
                                    <Link
                                        to={`/scraping/${result.scraping_id}`}
                                        className="text-xs font-bold px-4 py-2 bg-slate-900 border border-slate-700 rounded-lg text-slate-400 hover:text-white hover:border-slate-500 transition-all whitespace-nowrap"
                                    >
                                        View in Detail
                                    </Link>
                                </div>

                                <div className="space-y-3">
                                    {result.highlights.length > 0 ? (
                                        result.highlights.map((snippet, sIdx) => (
                                            <div
                                                key={sIdx}
                                                className="text-sm text-slate-400 leading-relaxed bg-slate-900/40 p-3 rounded-lg border border-slate-800/50 italic"
                                                dangerouslySetInnerHTML={{ __html: snippet.replace(/<em>/g, '<span class="text-sky-400 font-bold not-italic">').replace(/<\/em>/g, '</span>') }}
                                            />
                                        ))
                                    ) : (
                                        <p className="text-sm text-slate-500 italic">No snippets available.</p>
                                    )}
                                </div>
                            </div>
                        ))}

                        {results.length === 0 && !loading && (
                            <div className="text-center py-24 bg-slate-800/10 rounded-3xl border border-slate-800/50 border-dashed">
                                <div className="h-20 w-20 bg-slate-800/50 rounded-full flex items-center justify-center mx-auto mb-6">
                                    <SearchIcon className="h-10 w-10 text-slate-700" />
                                </div>
                                <h3 className="text-xl font-bold text-white mb-2">No results found</h3>
                                <p className="text-slate-400 max-w-sm mx-auto italic">
                                    Try adjusting your search terms or verify that the content has been indexed.
                                </p>
                            </div>
                        )}
                    </div>
                )}
            </main>
        </div>
    );
}
