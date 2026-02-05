import { useEffect, useState } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import axios from 'axios';
import { Loader2, ArrowLeft, ExternalLink, ImageIcon, FileText, RefreshCw, Search } from 'lucide-react';
import logo from '../assets/logo.png';

const API_Base = 'http://localhost:8000';

interface ScrapingData {
    status: string;
    scraping: any;
    data?: {
        url: string;
        summary: string | null;
        terms: { term: string; frequency: number }[];
        images: { url: string; explanation: string | null }[];
    }[];
}

export default function ScrapingDetail() {
    const { id } = useParams();
    const [data, setData] = useState<ScrapingData | null>(null);
    const [loading, setLoading] = useState(true);
    const [rerunning, setRerunning] = useState(false);
    const navigate = useNavigate();

    const fetchStatus = async () => {
        try {
            const apiKey = localStorage.getItem('api_key');
            // Ensure endpoint matches backend (Conversation 99e54c26 mentioned /scrape?scraping_id=${id})
            // But GEMINI.md says high performance validation and other things.
            // Let's use the one that was there before I broke it: /api/scrapings/${id}/ or /scrape?scraping_id=${id}
            // Dashboard uses /api/scrapings/
            const response = await axios.get(`${API_Base}/scrape?scraping_id=${id}`, {
                headers: { 'X-API-Key': apiKey }
            });
            setData(response.data);

            if (response.data.status === 'PENDING' || response.data.status === 'RUNNING') {
                setTimeout(fetchStatus, 3000);
            }
        } catch (err) {
            console.error('Failed to fetch scraping status', err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchStatus();
    }, [id]);

    const handleRerun = async () => {
        if (!data || !data.scraping) return;
        setRerunning(true);
        try {
            const apiKey = localStorage.getItem('api_key');
            const { url, depth } = data.scraping;
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

    if (loading && !data) {
        return (
            <div className="min-h-screen bg-slate-900 flex items-center justify-center">
                <Loader2 className="animate-spin h-10 w-10 text-sky-500" />
            </div>
        );
    }

    const status = data?.status || 'UNKNOWN';

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
                    </div>
                </div>
            </nav>

            <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
                {/* Job Info Header */}
                <div className="bg-gradient-to-br from-slate-900 to-slate-800 p-8 rounded-3xl border border-slate-700 shadow-2xl mb-10 overflow-hidden relative">
                    <div className="absolute top-0 right-0 p-8 opacity-10 blur-2xl">
                        <Search size={120} className="text-sky-500 rotate-12" />
                    </div>
                    <div className="relative z-10">
                        <h1 className="text-4xl font-extrabold text-white mb-2">Scraping Report</h1>
                        <p className="text-slate-400 font-mono text-sm">INTERNAL_JOB_ID: {id}</p>
                    </div>
                </div>

                {/* Content Grid */}
                {status === 'COMPLETED' && data?.data && (
                    <div className="grid grid-cols-1 gap-10">
                        {data.data.map((page, idx) => (
                            <div key={idx} className="bg-slate-900 border border-slate-800 rounded-3xl overflow-hidden shadow-xl ring-1 ring-white/5">
                                {/* Page URL Bar */}
                                <div className="bg-slate-800/50 px-6 py-4 flex items-center justify-between border-b border-slate-800">
                                    <h2 className="text-lg font-bold text-sky-400 truncate flex items-center gap-2">
                                        <ExternalLink size={18} />
                                        <a href={page.url} target="_blank" rel="noopener noreferrer" className="hover:underline">
                                            {page.url}
                                        </a>
                                    </h2>
                                </div>

                                <div className="p-8 space-y-12">
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

                                    {/* Terms Analysis */}
                                    {page.terms.length > 0 && (
                                        <section>
                                            <div className="flex items-center gap-2 mb-6 text-slate-300">
                                                <RefreshCw className="h-5 w-5 text-sky-400" />
                                                <h3 className="text-sm font-bold uppercase tracking-widest">Semantic Frequency Analysis</h3>
                                            </div>
                                            <div className="flex flex-wrap gap-2">
                                                {page.terms.slice(0, 20).map((term, t) => (
                                                    <div key={t} className="px-4 py-2 bg-slate-800 border border-slate-700 rounded-xl text-sm font-medium hover:border-sky-500/50 hover:bg-sky-500/5 transition-all cursor-default">
                                                        <span className="text-slate-200">{term.term}</span>
                                                        <span className="ml-2 px-1.5 py-0.5 bg-slate-900 text-sky-400 rounded-md text-[10px] font-bold">
                                                            {term.frequency}
                                                        </span>
                                                    </div>
                                                ))}
                                            </div>
                                        </section>
                                    )}
                                </div>
                            </div>
                        ))}
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
