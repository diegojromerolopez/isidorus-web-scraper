import { useEffect, useState, useCallback } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import axios from 'axios';
import { Loader2, ArrowLeft, ExternalLink, ImageIcon, FileText, RefreshCw, Search, Trash2, ChevronDown, ChevronRight } from 'lucide-react';
import logo from '../assets/logo.png';

const API_Base = 'http://localhost:8000';


interface ScrapedImage {
    url: string;
    explanation: string | null;
}

interface ScrapedPage {
    url: string;
    summary: string | null;
    images: ScrapedImage[];
}

interface Scraping {
    status: string;
    url: string;
    depth: number;
    links_count?: number;
    pages?: ScrapedPage[];
}

export default function ScrapingDetail() {
    const { id } = useParams();
    const [scraping, setScraping] = useState<Scraping | null>(null);
    const [loading, setLoading] = useState(true);
    const [rerunning, setRerunning] = useState(false);
    const [expandedPages, setExpandedPages] = useState<Set<number>>(new Set());
    const navigate = useNavigate();

    const togglePage = (index: number) => {
        setExpandedPages(prev => {
            const next = new Set(prev);
            if (next.has(index)) {
                next.delete(index);
            } else {
                next.add(index);
            }
            return next;
        });
    };

    const handleDelete = async () => {
        if (!window.confirm('Are you sure you want to delete this scraping job? This action will remove all related data from the database, DynamoDB, and S3.')) {
            return;
        }

        try {
            const apiKey = localStorage.getItem('api_key');
            await axios.delete(`${API_Base}/scraping/${id}`, {
                headers: { 'X-API-Key': apiKey }
            });
            navigate('/dashboard');
        } catch (error) {
            console.error('Failed to delete scraping', error);
            alert('Failed to delete scraping job.');
        }
    };

    const fetchStatus = useCallback(async () => {
        try {
            const apiKey = localStorage.getItem('api_key');
            // Ensure endpoint matches backend (Conversation 99e54c26 mentioned /scrape?scraping_id=${id})
            // But GEMINI.md says high performance validation and other things.
            // Let's use the one that was there before I broke it: /api/scrapings/${id}/ or /scrape?scraping_id=${id}
            // Dashboard uses /api/scrapings/
            const response = await axios.get(`${API_Base}/scraping/${id}`, {
                headers: { 'X-API-Key': apiKey }
            });

            console.log(response.data);

            setScraping(response.data.scraping);

            const currentStatus = response.data.scraping?.status;
            if (currentStatus === 'PENDING' || currentStatus === 'RUNNING') {
                setTimeout(fetchStatus, 3000);
            }
        } catch (err) {
            console.error('Failed to fetch scraping status', err);
        } finally {
            setLoading(false);
        }
    }, [id]);

    useEffect(() => {
        fetchStatus();
    }, [fetchStatus]);

    const handleRerun = async () => {
        if (!scraping) return;

        if (!window.confirm('Are you sure you want to re-run this scraping job with the same URL and depth?')) {
            return;
        }

        setRerunning(true);
        try {
            const apiKey = localStorage.getItem('api_key');
            const { url, depth } = scraping;
            const response = await axios.post(`${API_Base}/scrape`,
                { url, depth },
                { headers: { 'X-API-Key': apiKey } }
            );
            const newId = response.data.scraping_id;
            navigate(`/scraping/${newId}`);
        } catch (err) {
            console.error('Failed to re-run scraping', err);
        } finally {
            setRerunning(false);
        }
    };

    if (loading && !scraping) {
        return (
            <div className="min-h-screen bg-slate-900 flex items-center justify-center">
                <Loader2 className="animate-spin h-10 w-10 text-sky-500" />
            </div>
        );
    }

    const status = scraping?.status || 'UNKNOWN';

    return (
        <div className="min-h-screen bg-slate-900 text-slate-100 selection:bg-sky-500/30">
            {/* Header / Nav */}
            <nav className="border-b border-slate-800 bg-slate-900/50 backdrop-blur-md sticky top-0 z-50">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
                    <div className="flex items-center gap-6">
                        <Link to="/" className="p-2 -ml-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-all">
                            <ArrowLeft size={20} />
                        </Link>
                        <div className="flex items-center gap-3">
                            <div className="h-9 w-9 bg-sky-500/10 rounded-lg flex items-center justify-center border border-sky-500/20 overflow-hidden">
                                <img src={logo} alt="Isidorus" className="w-full h-full object-cover" />
                            </div>
                            <span className="text-xl font-bold tracking-tight bg-gradient-to-r from-sky-400 to-blue-500 bg-clip-text text-transparent">
                                Isidorus
                            </span>
                        </div>
                    </div>
                    <div className="flex items-center gap-3">
                        <span className={`px-4 py-1 rounded-full text-xs font-bold uppercase tracking-wider border ${status === 'COMPLETED'
                            ? 'bg-green-500/10 border-green-500/50 text-green-400'
                            : status === 'PENDING' || status === 'RUNNING'
                                ? 'bg-amber-500/10 border-amber-500/50 text-amber-400'
                                : 'bg-slate-500/10 border-slate-500/50 text-slate-400'
                            }`}>
                            {status}
                        </span>
                        <button
                            onClick={handleRerun}
                            disabled={rerunning}
                            className="p-2 bg-slate-800 hover:bg-slate-700 text-sky-400 rounded-lg border border-slate-700 transition-all disabled:opacity-50 active:scale-95 cursor-pointer"
                            title="Re-run Scrape"
                        >
                            {rerunning ? <Loader2 className="animate-spin h-5 w-5" /> : <RefreshCw className="h-5 w-5" />}
                        </button>
                        <button
                            onClick={handleDelete}
                            className="p-2 bg-slate-800 hover:bg-red-500/10 text-slate-400 hover:text-red-400 rounded-lg border border-slate-700 hover:border-red-500/50 transition-all active:scale-95 cursor-pointer"
                            title="Delete Scrape"
                        >
                            <Trash2 className="h-5 w-5" />
                        </button>
                    </div>
                </div>
            </nav>

            <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
                {/* Job Info Header */}
                <div className="bg-gradient-to-br from-slate-900 to-slate-800 p-8 rounded-3xl border border-slate-700 shadow-2xl mb-10 overflow-hidden relative">
                    <div className="absolute top-0 right-0 p-8 opacity-10 blur-2xl">
                        <Search size={120} className="text-sky-500 rotate-12" />
                    </div>
                    <div className="relative z-10 flex flex-col md:flex-row md:items-end justify-between gap-4">
                        <div>
                            <h1 className="text-4xl font-extrabold text-white mb-2">Scraping Report</h1>
                            <p className="text-slate-400 font-mono text-sm">INTERNAL_ID: {id}</p>
                        </div>
                        <div className="bg-slate-900/80 px-6 py-4 rounded-2xl border border-slate-700/50 backdrop-blur-md">
                            <p className="text-xs font-bold text-slate-500 uppercase tracking-widest mb-1">Total Links Found</p>
                            <p className="text-3xl font-black text-sky-400 font-mono">
                                {scraping?.links_count || 0}
                            </p>
                        </div>
                    </div>
                </div>

                {/* Content Grid */}
                {status === 'COMPLETED' && scraping?.pages && (
                    <div className="grid grid-cols-1 gap-10">
                        {scraping.pages.map((page, idx) => {
                            const isExpanded = expandedPages.has(idx);
                            return (
                                <div key={idx} className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden shadow-xl ring-1 ring-white/5 transition-all">
                                    {/* Page URL Bar / Toggle Header */}
                                    <button
                                        onClick={() => togglePage(idx)}
                                        className="w-full bg-slate-800/50 px-6 py-4 flex items-center justify-between border-b border-slate-800 hover:bg-slate-800 transition-colors group/header"
                                    >
                                        <div className="flex items-center gap-3 min-w-0">
                                            <div className="p-1.5 bg-slate-900 rounded-lg text-slate-500 group-hover/header:text-sky-400 transition-colors">
                                                {isExpanded ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
                                            </div>
                                            <h2 className="text-lg font-bold text-sky-400 truncate flex items-center gap-2">
                                                <ExternalLink size={18} className="shrink-0" />
                                                <span className="truncate">{page.url}</span>
                                            </h2>
                                        </div>
                                        <a
                                            href={page.url}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            onClick={(e) => e.stopPropagation()}
                                            className="px-3 py-1 bg-sky-500/10 text-sky-400 text-xs font-bold rounded-lg border border-sky-500/20 hover:bg-sky-500/20 transition-all opacity-0 group-hover/header:opacity-100 hidden sm:block"
                                        >
                                            Open External
                                        </a>
                                    </button>

                                    {isExpanded && (
                                        <div className="p-8 space-y-12 animate-in slide-in-from-top-4 duration-300">
                                            {/* Summary Section */}
                                            {page.summary && (
                                                <section>
                                                    <div className="flex items-center gap-2 mb-4 text-slate-300">
                                                        <FileText className="h-5 w-5 text-sky-400" />
                                                        <h3 className="text-sm font-bold uppercase tracking-widest">AI Intelligence Summary</h3>
                                                    </div>
                                                    <div className="bg-slate-950/50 p-6 rounded-2xl border border-slate-800 text-slate-300 leading-relaxed italic">
                                                        "{page.summary}"
                                                    </div>
                                                </section>
                                            )}

                                            {/* Images Gallery */}
                                            {page.images.length > 0 && (
                                                <section>
                                                    <div className="flex items-center gap-2 mb-6 text-slate-300">
                                                        <ImageIcon className="h-5 w-5 text-sky-400" />
                                                        <h3 className="text-sm font-bold uppercase tracking-widest">Extracted Visual Evidence</h3>
                                                    </div>
                                                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                                                        {page.images.map((img, i) => (
                                                            <div key={i} className="group bg-slate-800/30 border border-slate-800 rounded-2xl overflow-hidden hover:border-sky-500/50 transition-all duration-300">
                                                                <div className="aspect-[4/3] bg-slate-950 relative overflow-hidden">
                                                                    <img
                                                                        src={img.url}
                                                                        alt=""
                                                                        className="object-contain w-full h-full group-hover:scale-105 transition-transform duration-500"
                                                                    />
                                                                </div>
                                                                {img.explanation && (
                                                                    <div className="p-4 bg-slate-900/80">
                                                                        <p className="text-xs text-slate-400 leading-tight group-hover:text-slate-200 transition-colors">
                                                                            {img.explanation}
                                                                        </p>
                                                                    </div>
                                                                )}
                                                            </div>
                                                        ))}
                                                    </div>
                                                </section>
                                            )}

                                        </div>
                                    )}
                                </div>
                            );
                        })}
                    </div>
                )}

                {status === 'PENDING' && (
                    <div className="flex flex-col items-center justify-center py-20 bg-slate-800/30 rounded-3xl border border-slate-800 dashed animate-pulse">
                        <Loader2 className="animate-spin h-10 w-10 text-sky-500 mb-4" />
                        <p className="text-slate-400">Scraping in progress... Please wait.</p>
                    </div>
                )}
            </main>
        </div>
    );
}
