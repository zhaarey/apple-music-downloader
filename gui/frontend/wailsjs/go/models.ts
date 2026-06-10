export namespace apple {
	
	export class PreviewTrack {
	    num: number;
	    id: string;
	    name: string;
	    artist: string;
	    type: string;
	    duration: string;
	    duration_ms: number;
	    explicit: boolean;
	    is_mv: boolean;
	    url?: string;
	    art_url?: string;
	    album?: string;
	    album_artist?: string;
	    genre?: string;
	    year?: string;
	    track_number?: number;
	    disc_number?: number;
	    on_disk?: boolean;
	    existing_path?: string;
	    existing_root_label?: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewTrack(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.num = source["num"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.artist = source["artist"];
	        this.type = source["type"];
	        this.duration = source["duration"];
	        this.duration_ms = source["duration_ms"];
	        this.explicit = source["explicit"];
	        this.is_mv = source["is_mv"];
	        this.url = source["url"];
	        this.art_url = source["art_url"];
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.track_number = source["track_number"];
	        this.disc_number = source["disc_number"];
	        this.on_disk = source["on_disk"];
	        this.existing_path = source["existing_path"];
	        this.existing_root_label = source["existing_root_label"];
	    }
	}
	export class PreviewResult {
	    url: string;
	    type: string;
	    title: string;
	    subtitle: string;
	    art_url: string;
	    track_count: number;
	    total_duration: string;
	    tracks: PreviewTrack[];
	    can_select_tracks: boolean;
	    output_folder: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.type = source["type"];
	        this.title = source["title"];
	        this.subtitle = source["subtitle"];
	        this.art_url = source["art_url"];
	        this.track_count = source["track_count"];
	        this.total_duration = source["total_duration"];
	        this.tracks = this.convertValues(source["tracks"], PreviewTrack);
	        this.can_select_tracks = source["can_select_tracks"];
	        this.output_folder = source["output_folder"];
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace engine {
	
	export class BulkQueueEntry {
	    url: string;
	    selected_track_nums: number[];
	    force_track_nums: number[];
	
	    static createFrom(source: any = {}) {
	        return new BulkQueueEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.selected_track_nums = source["selected_track_nums"];
	        this.force_track_nums = source["force_track_nums"];
	    }
	}
	export class DependencyStatus {
	    name: string;
	    ok: boolean;
	    detail: string;
	    required: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DependencyStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ok = source["ok"];
	        this.detail = source["detail"];
	        this.required = source["required"];
	    }
	}
	export class TrackDuplicateStatus {
	    num: number;
	    on_disk: boolean;
	    existing_path?: string;
	    existing_root_label?: string;
	    expected_path?: string;
	    expected_filename?: string;
	
	    static createFrom(source: any = {}) {
	        return new TrackDuplicateStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.num = source["num"];
	        this.on_disk = source["on_disk"];
	        this.existing_path = source["existing_path"];
	        this.existing_root_label = source["existing_root_label"];
	        this.expected_path = source["expected_path"];
	        this.expected_filename = source["expected_filename"];
	    }
	}
	export class DuplicateCheckResult {
	    roots: string[];
	    tracks: TrackDuplicateStatus[];
	    existing_count: number;
	    selected_count: number;
	
	    static createFrom(source: any = {}) {
	        return new DuplicateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.roots = source["roots"];
	        this.tracks = this.convertValues(source["tracks"], TrackDuplicateStatus);
	        this.existing_count = source["existing_count"];
	        this.selected_count = source["selected_count"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PreflightCheck {
	    id: string;
	    label: string;
	    ok: boolean;
	    detail: string;
	    blocking: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PreflightCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.ok = source["ok"];
	        this.detail = source["detail"];
	        this.blocking = source["blocking"];
	    }
	}
	export class PreflightResult {
	    ready: boolean;
	    summary: string;
	    checks: PreflightCheck[];
	
	    static createFrom(source: any = {}) {
	        return new PreflightResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.summary = source["summary"];
	        this.checks = this.convertValues(source["checks"], PreflightCheck);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SearchHit {
	    type: string;
	    name: string;
	    detail: string;
	    url: string;
	    id: string;
	    art_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.detail = source["detail"];
	        this.url = source["url"];
	        this.id = source["id"];
	        this.art_url = source["art_url"];
	    }
	}
	export class SpotifyMatchItem {
	    spotify_title: string;
	    spotify_artist: string;
	    spotify_album?: string;
	    spotify_isrc?: string;
	    match_status: string;
	    match_method?: string;
	    score: number;
	    apple_hit?: SearchHit;
	    alternatives?: SearchHit[];
	
	    static createFrom(source: any = {}) {
	        return new SpotifyMatchItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.spotify_title = source["spotify_title"];
	        this.spotify_artist = source["spotify_artist"];
	        this.spotify_album = source["spotify_album"];
	        this.spotify_isrc = source["spotify_isrc"];
	        this.match_status = source["match_status"];
	        this.match_method = source["match_method"];
	        this.score = source["score"];
	        this.apple_hit = this.convertValues(source["apple_hit"], SearchHit);
	        this.alternatives = this.convertValues(source["alternatives"], SearchHit);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SpotifyResolveResult {
	    source_kind: string;
	    source_title: string;
	    source_url: string;
	    track_count: number;
	    matched: number;
	    isrc_matched: number;
	    uncertain: number;
	    missing: number;
	    items: SpotifyMatchItem[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SpotifyResolveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_kind = source["source_kind"];
	        this.source_title = source["source_title"];
	        this.source_url = source["source_url"];
	        this.track_count = source["track_count"];
	        this.matched = source["matched"];
	        this.isrc_matched = source["isrc_matched"];
	        this.uncertain = source["uncertain"];
	        this.missing = source["missing"];
	        this.items = this.convertValues(source["items"], SpotifyMatchItem);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class TagDropResolve {
	    mode: string;
	    path: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new TagDropResolve(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.path = source["path"];
	        this.message = source["message"];
	    }
	}

}

export namespace media {
	
	export class AudioTagInfo {
	    path: string;
	    title: string;
	    artist: string;
	    album: string;
	    album_artist: string;
	    genre: string;
	    year: string;
	    track_number: number;
	    track_total: number;
	    disc_number: number;
	    disc_total: number;
	    has_artwork: boolean;
	    artwork_count: number;
	    artwork_mime?: string;
	    artwork_b64?: string;
	    summary: string;
	    media_kind?: string;
	    video_codec?: string;
	    audio_codec?: string;
	    video_width?: number;
	    video_height?: number;
	    duration_label?: string;
	    apple_video_ready?: boolean;
	    apple_video_detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new AudioTagInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.track_number = source["track_number"];
	        this.track_total = source["track_total"];
	        this.disc_number = source["disc_number"];
	        this.disc_total = source["disc_total"];
	        this.has_artwork = source["has_artwork"];
	        this.artwork_count = source["artwork_count"];
	        this.artwork_mime = source["artwork_mime"];
	        this.artwork_b64 = source["artwork_b64"];
	        this.summary = source["summary"];
	        this.media_kind = source["media_kind"];
	        this.video_codec = source["video_codec"];
	        this.audio_codec = source["audio_codec"];
	        this.video_width = source["video_width"];
	        this.video_height = source["video_height"];
	        this.duration_label = source["duration_label"];
	        this.apple_video_ready = source["apple_video_ready"];
	        this.apple_video_detail = source["apple_video_detail"];
	    }
	}
	export class AlbumFolderReadResult {
	    tracks: AudioTagInfo[];
	    skipped?: string[];
	
	    static createFrom(source: any = {}) {
	        return new AlbumFolderReadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tracks = this.convertValues(source["tracks"], AudioTagInfo);
	        this.skipped = source["skipped"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AlbumPreparePreview {
	    folder: string;
	    track_count: number;
	    cover_source: string;
	    recursive: boolean;
	    warning: string;
	
	    static createFrom(source: any = {}) {
	        return new AlbumPreparePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = source["folder"];
	        this.track_count = source["track_count"];
	        this.cover_source = source["cover_source"];
	        this.recursive = source["recursive"];
	        this.warning = source["warning"];
	    }
	}
	export class AlbumPrepareResult {
	    folder: string;
	    prepared: number;
	    skipped: number;
	    errors: string[];
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new AlbumPrepareResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = source["folder"];
	        this.prepared = source["prepared"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	        this.summary = source["summary"];
	    }
	}
	export class AppleMusicCacheInfo {
	    paths: string[];
	    platform: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new AppleMusicCacheInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paths = source["paths"];
	        this.platform = source["platform"];
	        this.note = source["note"];
	    }
	}
	export class SyncRepairStep {
	    id: string;
	    label: string;
	    ok: boolean;
	    detail: string;
	    skipped: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SyncRepairStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.ok = source["ok"];
	        this.detail = source["detail"];
	        this.skipped = source["skipped"];
	    }
	}
	export class ApplePurgeResult {
	    ok: boolean;
	    summary: string;
	    message: string;
	    steps: SyncRepairStep[];
	    log_path: string;
	    need_elevated: boolean;
	    manual_steps: string[];
	
	    static createFrom(source: any = {}) {
	        return new ApplePurgeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.summary = source["summary"];
	        this.message = source["message"];
	        this.steps = this.convertValues(source["steps"], SyncRepairStep);
	        this.log_path = source["log_path"];
	        this.need_elevated = source["need_elevated"];
	        this.manual_steps = source["manual_steps"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppleSyncUnlockResult {
	    ok: boolean;
	    summary: string;
	    message: string;
	    log_path: string;
	    need_elevated: boolean;
	    manual_steps: string[];
	    killed_hint?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppleSyncUnlockResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.summary = source["summary"];
	        this.message = source["message"];
	        this.log_path = source["log_path"];
	        this.need_elevated = source["need_elevated"];
	        this.manual_steps = source["manual_steps"];
	        this.killed_hint = source["killed_hint"];
	    }
	}
	export class ArtworkAccentAnalysis {
	    width: number;
	    height: number;
	    is_square: boolean;
	    min_edge_px: number;
	    avg_saturation: number;
	    avg_luminance: number;
	    accent_ready: boolean;
	    warnings: string[];
	    summary: string;
	    optimized_b64?: string;
	    optimized_mime?: string;
	
	    static createFrom(source: any = {}) {
	        return new ArtworkAccentAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.width = source["width"];
	        this.height = source["height"];
	        this.is_square = source["is_square"];
	        this.min_edge_px = source["min_edge_px"];
	        this.avg_saturation = source["avg_saturation"];
	        this.avg_luminance = source["avg_luminance"];
	        this.accent_ready = source["accent_ready"];
	        this.warnings = source["warnings"];
	        this.summary = source["summary"];
	        this.optimized_b64 = source["optimized_b64"];
	        this.optimized_mime = source["optimized_mime"];
	    }
	}
	
	export class CacheClearResult {
	    ok: boolean;
	    message: string;
	    cleared: string[];
	    errors: string[];
	    platform: string;
	
	    static createFrom(source: any = {}) {
	        return new CacheClearResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.cleared = source["cleared"];
	        this.errors = source["errors"];
	        this.platform = source["platform"];
	    }
	}
	export class SyncValidationResult {
	    path: string;
	    ready: boolean;
	    summary: string;
	    checks: SyncCheck[];
	
	    static createFrom(source: any = {}) {
	        return new SyncValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.ready = source["ready"];
	        this.summary = source["summary"];
	        this.checks = this.convertValues(source["checks"], SyncCheck);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SyncCheck {
	    id: string;
	    label: string;
	    pass: boolean;
	    detail: string;
	    severity: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.pass = source["pass"];
	        this.detail = source["detail"];
	        this.severity = source["severity"];
	    }
	}
	export class FolderSyncValidationResult {
	    folder: string;
	    ready: boolean;
	    total: number;
	    ready_count: number;
	    summary: string;
	    folder_checks?: SyncCheck[];
	    files: SyncValidationResult[];
	
	    static createFrom(source: any = {}) {
	        return new FolderSyncValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = source["folder"];
	        this.ready = source["ready"];
	        this.total = source["total"];
	        this.ready_count = source["ready_count"];
	        this.summary = source["summary"];
	        this.folder_checks = this.convertValues(source["folder_checks"], SyncCheck);
	        this.files = this.convertValues(source["files"], SyncValidationResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PreparedArtworkResult {
	    path: string;
	    width: number;
	    height: number;
	    is_square: boolean;
	    min_edge_px: number;
	    avg_saturation: number;
	    avg_luminance: number;
	    accent_ready: boolean;
	    warnings: string[];
	    summary: string;
	    optimized_b64?: string;
	    optimized_mime?: string;
	
	    static createFrom(source: any = {}) {
	        return new PreparedArtworkResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.is_square = source["is_square"];
	        this.min_edge_px = source["min_edge_px"];
	        this.avg_saturation = source["avg_saturation"];
	        this.avg_luminance = source["avg_luminance"];
	        this.accent_ready = source["accent_ready"];
	        this.warnings = source["warnings"];
	        this.summary = source["summary"];
	        this.optimized_b64 = source["optimized_b64"];
	        this.optimized_mime = source["optimized_mime"];
	    }
	}
	
	export class SyncRepairOptions {
	    prepare_folders: string[];
	    skip_prepare: boolean;
	    force_if_music_running: boolean;
	    cache_only: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SyncRepairOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prepare_folders = source["prepare_folders"];
	        this.skip_prepare = source["skip_prepare"];
	        this.force_if_music_running = source["force_if_music_running"];
	        this.cache_only = source["cache_only"];
	    }
	}
	export class SyncRepairPreparePreview {
	    folders: string[];
	    track_count: number;
	    warning: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncRepairPreparePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folders = source["folders"];
	        this.track_count = source["track_count"];
	        this.warning = source["warning"];
	    }
	}
	export class SyncRepairResult {
	    ok: boolean;
	    summary: string;
	    steps: SyncRepairStep[];
	    need_elevated: boolean;
	    log_path: string;
	    manual_steps: string[];
	
	    static createFrom(source: any = {}) {
	        return new SyncRepairResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.summary = source["summary"];
	        this.steps = this.convertValues(source["steps"], SyncRepairStep);
	        this.need_elevated = source["need_elevated"];
	        this.log_path = source["log_path"];
	        this.manual_steps = source["manual_steps"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class TagAlbumTrackInput {
	    path: string;
	    title: string;
	    artist: string;
	    track_number: number;
	    disc_number: number;
	
	    static createFrom(source: any = {}) {
	        return new TagAlbumTrackInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.track_number = source["track_number"];
	        this.disc_number = source["disc_number"];
	    }
	}
	export class TagAlbumBatchInput {
	    folder: string;
	    album: string;
	    album_artist: string;
	    genre: string;
	    year: string;
	    track_total: number;
	    disc_total: number;
	    cover_path: string;
	    clear_artwork: boolean;
	    sort_tags: boolean;
	    optimize_artwork?: boolean;
	    write_cover_sidecar?: boolean;
	    mp4box_reembed: boolean;
	    tracks: TagAlbumTrackInput[];
	
	    static createFrom(source: any = {}) {
	        return new TagAlbumBatchInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = source["folder"];
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.track_total = source["track_total"];
	        this.disc_total = source["disc_total"];
	        this.cover_path = source["cover_path"];
	        this.clear_artwork = source["clear_artwork"];
	        this.sort_tags = source["sort_tags"];
	        this.optimize_artwork = source["optimize_artwork"];
	        this.write_cover_sidecar = source["write_cover_sidecar"];
	        this.mp4box_reembed = source["mp4box_reembed"];
	        this.tracks = this.convertValues(source["tracks"], TagAlbumTrackInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TagAlbumBatchResult {
	    saved: number;
	    errors: string[];
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new TagAlbumBatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.saved = source["saved"];
	        this.errors = source["errors"];
	        this.summary = source["summary"];
	    }
	}
	
	export class WriteAudioTagsInput {
	    path: string;
	    title: string;
	    artist: string;
	    album: string;
	    album_artist: string;
	    genre: string;
	    year: string;
	    track_number: number;
	    track_total: number;
	    disc_number: number;
	    disc_total: number;
	    cover_path: string;
	    clear_artwork: boolean;
	    sort_tags: boolean;
	    optimize_artwork?: boolean;
	    write_cover_sidecar?: boolean;
	    mp4box_reembed: boolean;
	    output_path?: string;
	
	    static createFrom(source: any = {}) {
	        return new WriteAudioTagsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.track_number = source["track_number"];
	        this.track_total = source["track_total"];
	        this.disc_number = source["disc_number"];
	        this.disc_total = source["disc_total"];
	        this.cover_path = source["cover_path"];
	        this.clear_artwork = source["clear_artwork"];
	        this.sort_tags = source["sort_tags"];
	        this.optimize_artwork = source["optimize_artwork"];
	        this.write_cover_sidecar = source["write_cover_sidecar"];
	        this.mp4box_reembed = source["mp4box_reembed"];
	        this.output_path = source["output_path"];
	    }
	}

}

export namespace splice {
	
	export class AlbumMetadata {
	    album: string;
	    album_artist: string;
	    artist: string;
	    year: string;
	    genre: string;
	    artwork_path?: string;
	    total_tracks?: number;
	
	    static createFrom(source: any = {}) {
	        return new AlbumMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.artist = source["artist"];
	        this.year = source["year"];
	        this.genre = source["genre"];
	        this.artwork_path = source["artwork_path"];
	        this.total_tracks = source["total_tracks"];
	    }
	}
	export class MasterProbe {
	    duration_ms: number;
	    sample_rate: number;
	    channels: number;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new MasterProbe(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.duration_ms = source["duration_ms"];
	        this.sample_rate = source["sample_rate"];
	        this.channels = source["channels"];
	        this.summary = source["summary"];
	    }
	}
	export class PeakBin {
	    min: number;
	    max: number;
	
	    static createFrom(source: any = {}) {
	        return new PeakBin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.min = source["min"];
	        this.max = source["max"];
	    }
	}
	export class Track {
	    title: string;
	    duration_ms: number;
	    start_ms?: number;
	    artist?: string;
	    track_number?: number;
	    duration?: string;
	    album?: string;
	    album_artist?: string;
	    genre?: string;
	    year?: string;
	    disc_number?: number;
	    disc_total?: number;
	
	    static createFrom(source: any = {}) {
	        return new Track(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.duration_ms = source["duration_ms"];
	        this.start_ms = source["start_ms"];
	        this.artist = source["artist"];
	        this.track_number = source["track_number"];
	        this.duration = source["duration"];
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.disc_number = source["disc_number"];
	        this.disc_total = source["disc_total"];
	    }
	}
	export class Project {
	    master_path: string;
	    output_dir: string;
	    album: AlbumMetadata;
	    tracks: Track[];
	    master_duration_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.master_path = source["master_path"];
	        this.output_dir = source["output_dir"];
	        this.album = this.convertValues(source["album"], AlbumMetadata);
	        this.tracks = this.convertValues(source["tracks"], Track);
	        this.master_duration_ms = source["master_duration_ms"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class WaveformPeaks {
	    bins: PeakBin[];
	    duration_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new WaveformPeaks(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bins = this.convertValues(source["bins"], PeakBin);
	        this.duration_ms = source["duration_ms"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace structs {
	
	export class ConfigSet {
	    storefront: string;
	    "media-user-token": string;
	    "authorization-token": string;
	    language: string;
	    "save-lrc-file": boolean;
	    "lrc-type": string;
	    "lrc-format": string;
	    "save-animated-artwork": boolean;
	    "emby-animated-artwork": boolean;
	    "embed-lrc": boolean;
	    "embed-cover": boolean;
	    "save-artist-cover": boolean;
	    "cover-size": string;
	    "cover-format": string;
	    "tag-sort-order": boolean;
	    "tag-itunes-id": boolean;
	    "alac-save-folder": string;
	    "atmos-save-folder": string;
	    "aac-save-folder": string;
	    "mv-save-folder": string;
	    "album-folder-format": string;
	    "playlist-folder-format": string;
	    "artist-folder-format": string;
	    "song-file-format": string;
	    "explicit-choice": string;
	    "clean-choice": string;
	    "apple-master-choice": string;
	    "max-memory-limit": number;
	    "decrypt-m3u8-port": string;
	    "get-m3u8-port": string;
	    "get-m3u8-mode": string;
	    "get-m3u8-from-device": boolean;
	    "aac-type": string;
	    "alac-max": number;
	    "atmos-max": number;
	    "limit-max": number;
	    "use-songinfo-for-playlist": boolean;
	    "dl-albumcover-for-playlist": boolean;
	    "mv-audio-type": string;
	    "mv-max": number;
	    "convert-after-download": boolean;
	    "convert-format": string;
	    "convert-keep-original": boolean;
	    "convert-skip-if-source-matches": boolean;
	    "ffmpeg-path": string;
	    "convert-extra-args": string;
	    "convert-with-metadata": boolean;
	    "convert-warn-lossy-to-lossless": boolean;
	    "convert-skip-lossy-to-lossless": boolean;
	    "convert-check-bad-alac": boolean;
	    "convert-delete-bad-alac": boolean;
	    "alac-fix": boolean;
	    "exit-on-error": boolean;
	    "youtube-mode": boolean;
	    "yt-dlp-path": string;
	    "youtube-output-location": string;
	    "youtube-save-folder": string;
	    "spotify-client-id": string;
	    "spotify-client-secret": string;
	    "duplicate-check-folders": string[];
	
	    static createFrom(source: any = {}) {
	        return new ConfigSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.storefront = source["storefront"];
	        this["media-user-token"] = source["media-user-token"];
	        this["authorization-token"] = source["authorization-token"];
	        this.language = source["language"];
	        this["save-lrc-file"] = source["save-lrc-file"];
	        this["lrc-type"] = source["lrc-type"];
	        this["lrc-format"] = source["lrc-format"];
	        this["save-animated-artwork"] = source["save-animated-artwork"];
	        this["emby-animated-artwork"] = source["emby-animated-artwork"];
	        this["embed-lrc"] = source["embed-lrc"];
	        this["embed-cover"] = source["embed-cover"];
	        this["save-artist-cover"] = source["save-artist-cover"];
	        this["cover-size"] = source["cover-size"];
	        this["cover-format"] = source["cover-format"];
	        this["tag-sort-order"] = source["tag-sort-order"];
	        this["tag-itunes-id"] = source["tag-itunes-id"];
	        this["alac-save-folder"] = source["alac-save-folder"];
	        this["atmos-save-folder"] = source["atmos-save-folder"];
	        this["aac-save-folder"] = source["aac-save-folder"];
	        this["mv-save-folder"] = source["mv-save-folder"];
	        this["album-folder-format"] = source["album-folder-format"];
	        this["playlist-folder-format"] = source["playlist-folder-format"];
	        this["artist-folder-format"] = source["artist-folder-format"];
	        this["song-file-format"] = source["song-file-format"];
	        this["explicit-choice"] = source["explicit-choice"];
	        this["clean-choice"] = source["clean-choice"];
	        this["apple-master-choice"] = source["apple-master-choice"];
	        this["max-memory-limit"] = source["max-memory-limit"];
	        this["decrypt-m3u8-port"] = source["decrypt-m3u8-port"];
	        this["get-m3u8-port"] = source["get-m3u8-port"];
	        this["get-m3u8-mode"] = source["get-m3u8-mode"];
	        this["get-m3u8-from-device"] = source["get-m3u8-from-device"];
	        this["aac-type"] = source["aac-type"];
	        this["alac-max"] = source["alac-max"];
	        this["atmos-max"] = source["atmos-max"];
	        this["limit-max"] = source["limit-max"];
	        this["use-songinfo-for-playlist"] = source["use-songinfo-for-playlist"];
	        this["dl-albumcover-for-playlist"] = source["dl-albumcover-for-playlist"];
	        this["mv-audio-type"] = source["mv-audio-type"];
	        this["mv-max"] = source["mv-max"];
	        this["convert-after-download"] = source["convert-after-download"];
	        this["convert-format"] = source["convert-format"];
	        this["convert-keep-original"] = source["convert-keep-original"];
	        this["convert-skip-if-source-matches"] = source["convert-skip-if-source-matches"];
	        this["ffmpeg-path"] = source["ffmpeg-path"];
	        this["convert-extra-args"] = source["convert-extra-args"];
	        this["convert-with-metadata"] = source["convert-with-metadata"];
	        this["convert-warn-lossy-to-lossless"] = source["convert-warn-lossy-to-lossless"];
	        this["convert-skip-lossy-to-lossless"] = source["convert-skip-lossy-to-lossless"];
	        this["convert-check-bad-alac"] = source["convert-check-bad-alac"];
	        this["convert-delete-bad-alac"] = source["convert-delete-bad-alac"];
	        this["alac-fix"] = source["alac-fix"];
	        this["exit-on-error"] = source["exit-on-error"];
	        this["youtube-mode"] = source["youtube-mode"];
	        this["yt-dlp-path"] = source["yt-dlp-path"];
	        this["youtube-output-location"] = source["youtube-output-location"];
	        this["youtube-save-folder"] = source["youtube-save-folder"];
	        this["spotify-client-id"] = source["spotify-client-id"];
	        this["spotify-client-secret"] = source["spotify-client-secret"];
	        this["duplicate-check-folders"] = source["duplicate-check-folders"];
	    }
	}

}

export namespace trim {
	
	export class ExportInput {
	    source_path: string;
	    output_path: string;
	    start_ms: number;
	    end_ms: number;
	    copy_tags: boolean;
	    overwrite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExportInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_path = source["source_path"];
	        this.output_path = source["output_path"];
	        this.start_ms = source["start_ms"];
	        this.end_ms = source["end_ms"];
	        this.copy_tags = source["copy_tags"];
	        this.overwrite = source["overwrite"];
	    }
	}
	export class ProbeResult {
	    duration_ms: number;
	    has_video: boolean;
	    has_audio: boolean;
	    summary: string;
	    media_kind: string;
	
	    static createFrom(source: any = {}) {
	        return new ProbeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.duration_ms = source["duration_ms"];
	        this.has_video = source["has_video"];
	        this.has_audio = source["has_audio"];
	        this.summary = source["summary"];
	        this.media_kind = source["media_kind"];
	    }
	}

}

export namespace youtube {
	
	export class DownloadMeta {
	    num: number;
	    title: string;
	    artist: string;
	    album: string;
	    album_artist: string;
	    genre: string;
	    year: string;
	    track_number: number;
	    disc_number: number;
	    track_total: number;
	    art_url?: string;
	    cover_path?: string;
	    art_source?: string;
	    optimize_artwork?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DownloadMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.num = source["num"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.album_artist = source["album_artist"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.track_number = source["track_number"];
	        this.disc_number = source["disc_number"];
	        this.track_total = source["track_total"];
	        this.art_url = source["art_url"];
	        this.cover_path = source["cover_path"];
	        this.art_source = source["art_source"];
	        this.optimize_artwork = source["optimize_artwork"];
	    }
	}

}

