import { useEffect, useState } from 'react';
import { useNavigate, useParams, Link } from 'react-router-dom';
import axios from 'axios';
import { Loader2, ArrowLeft, ExternalLink, ImageIcon, FileText, RefreshCw, Search } from 'lucide-react';

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

    const handleRerun = async () => {
        if (!data || !data.scraping) return;
        setRerunning(true);
        try {
            const apiKey = localStorage.getItem('api_key');
            const { url, depth } = data.scraping;

            // Default depth to 1 if missing (for legacy jobs)
            const response = await axios.post(
                `${API_Base}/scrape`,
                { url, depth: depth || 1 },
                { headers: { 'X-API-Key': apiKey } }
            );

            navigate(`/scraping/${response.data.scraping_id}`);
        } catch (error) {
            console.error('Failed to rerun scraping', error);
        } finally {
            setRerunning(false);
        }
    };

    useEffect(() => {
        let interval: number;

        const fetchStatus = async () => {
            try {
                const apiKey = localStorage.getItem('api_key');
                const response = await axios.get(`${API_Base}/scrape?scraping_id=${id}`, {
                    headers: { 'X-API-Key': apiKey },
                });
                setData(response.data);

                if (response.data.status === 'COMPLETED') {
                    setLoading(false);
                    clearInterval(interval);
                }
            } catch (error) {
                console.error('Failed to fetch scraping status', error);
                setLoading(false); // Stop trying blindly on error
                clearInterval(interval);
            }
        };

        fetchStatus();
        interval = setInterval(fetchStatus, 2000); // Poll every 2s

        return () => clearInterval(interval);
    }, [id]);

    if (!data && loading) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <Loader2 className="animate-spin h-8 w-8 text-blue-600" />
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-slate-950 text-slate-100 selection:bg-sky-500/30 pb-20">
            {/* Top Navigation */}
            <nav className="border-b border-slate-800 bg-slate-900/50 backdrop-blur-md sticky top-0 z-50">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
                    <Link to="/" className="flex items-center gap-2 text-slate-400 hover:text-white transition-colors group">
                        <ArrowLeft className="h-5 w-5 group-hover:-translate-x-1 transition-transform" />
                        <span className="font-medium text-sm">Back to Dashboard</span>
                    </Link>
                    <div className="flex items-center gap-3">
                        <span className={`px-4 py-1 rounded-full text-xs font-bold uppercase tracking-wider border ${data?.status === 'COMPLETED'
                            ? 'bg-green-500/10 border-green-500/50 text-green-400'
                            : data?.status === 'PENDING'
                                ? 'bg-amber-500/10 border-amber-500/50 text-amber-400'
                                : 'bg-slate-500/10 border-slate-500/50 text-slate-400'
                            }`}>
                            {data?.status || 'UNKNOWN'}
                        </span>
                        <button
                            onClick={handleRerun}
                            disabled={rerunning}
                            className="p-2 bg-slate-800 hover:bg-slate-700 text-sky-400 rounded-lg border border-slate-700 transition-all disabled:opacity-50 active:scale-95"
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
                {data?.status === 'COMPLETED' && data.data && (
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
            </main>
        </div>
    );
}

