export namespace main {
	
	export class Event {
	    id: string;
	    name: string;
	    type: string;
	    start: string;
	    end: string;
	    city: string;
	    venue: string;
	    age: string;
	    features: string;
	    comment: string;
	    status: string;
	    sportStatus: string;
	    category: string;
	    sheet: string;
	
	    static createFrom(source: any = {}) {
	        return new Event(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.start = source["start"];
	        this.end = source["end"];
	        this.city = source["city"];
	        this.venue = source["venue"];
	        this.age = source["age"];
	        this.features = source["features"];
	        this.comment = source["comment"];
	        this.status = source["status"];
	        this.sportStatus = source["sportStatus"];
	        this.category = source["category"];
	        this.sheet = source["sheet"];
	    }
	}
	export class Season {
	    sy: number;
	    sm: number;
	    ey: number;
	    em: number;
	
	    static createFrom(source: any = {}) {
	        return new Season(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sy = source["sy"];
	        this.sm = source["sm"];
	        this.ey = source["ey"];
	        this.em = source["em"];
	    }
	}
	export class State {
	    events: Event[];
	    season: Season;
	    today: string;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.events = this.convertValues(source["events"], Event);
	        this.season = this.convertValues(source["season"], Season);
	        this.today = source["today"];
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

