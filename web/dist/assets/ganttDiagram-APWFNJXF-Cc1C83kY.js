import{_ as c,d as j,e as ce,s as ct,g as lt,o as ut,p as dt,c as ft,b as ht,v as kt,n as mt,l as Te,k as ge,aw as yt,ax as gt,ay as pt,m as vt,S as Tt,az as xt,aA as bt,aB as Ne,aC as Be,aD as Ge,aE as He,aF as Xe,aG as je,aH as qe,aI as wt,f as _t,u as Dt,aJ as St,aK as Et,aL as Ct,aM as Mt,aN as It,aO as At,aP as Lt}from"./mermaid-9C29qDeS.js";import{g as Ce,c as Me}from"./react-core-CZZu3mg-.js";var Ke={exports:{}};(function(e,r){(function(n,a){e.exports=a()})(Me,function(){return function(n,a){var i=a.prototype,k=i.format;i.format=function(f){var _=this,Y=this.$locale();if(!this.isValid())return k.bind(this)(f);var C=this.$utils(),g=(f||"YYYY-MM-DDTHH:mm:ssZ").replace(/\[([^\]]+)]|Q|wo|ww|w|WW|W|zzz|z|gggg|GGGG|Do|X|x|k{1,2}|S/g,function(M){switch(M){case"Q":return Math.ceil((_.$M+1)/3);case"Do":return Y.ordinal(_.$D);case"gggg":return _.weekYear();case"GGGG":return _.isoWeekYear();case"wo":return Y.ordinal(_.week(),"W");case"w":case"ww":return C.s(_.week(),M==="w"?1:2,"0");case"W":case"WW":return C.s(_.isoWeek(),M==="W"?1:2,"0");case"k":case"kk":return C.s(String(_.$H===0?24:_.$H),M==="k"?1:2,"0");case"X":return Math.floor(_.$d.getTime()/1e3);case"x":return _.$d.getTime();case"z":return"["+_.offsetName()+"]";case"zzz":return"["+_.offsetName("long")+"]";default:return M}});return k.bind(this)(g)}}})})(Ke);var Ft=Ke.exports;const Yt=Ce(Ft);var Je={exports:{}};(function(e,r){(function(n,a){e.exports=a()})(Me,function(){var n={LTS:"h:mm:ss A",LT:"h:mm A",L:"MM/DD/YYYY",LL:"MMMM D, YYYY",LLL:"MMMM D, YYYY h:mm A",LLLL:"dddd, MMMM D, YYYY h:mm A"},a=/(\[[^[]*\])|([-_:/.,()\s]+)|(A|a|Q|YYYY|YY?|ww?|MM?M?M?|Do|DD?|hh?|HH?|mm?|ss?|S{1,3}|z|ZZ?)/g,i=/\d/,k=/\d\d/,f=/\d\d?/,_=/\d*[^-_:/,()\s\d]+/,Y={},C=function(p){return(p=+p)+(p>68?1900:2e3)},g=function(p){return function(S){this[p]=+S}},M=[/[+-]\d\d:?(\d\d)?|Z/,function(p){(this.zone||(this.zone={})).offset=function(S){if(!S||S==="Z")return 0;var L=S.match(/([+-]|\d\d)/g),F=60*L[1]+(+L[2]||0);return F===0?0:L[0]==="+"?-F:F}(p)}],V=function(p){var S=Y[p];return S&&(S.indexOf?S:S.s.concat(S.f))},P=function(p,S){var L,F=Y.meridiem;if(F){for(var H=1;H<=24;H+=1)if(p.indexOf(F(H,0,S))>-1){L=H>12;break}}else L=p===(S?"pm":"PM");return L},B={A:[_,function(p){this.afternoon=P(p,!1)}],a:[_,function(p){this.afternoon=P(p,!0)}],Q:[i,function(p){this.month=3*(p-1)+1}],S:[i,function(p){this.milliseconds=100*+p}],SS:[k,function(p){this.milliseconds=10*+p}],SSS:[/\d{3}/,function(p){this.milliseconds=+p}],s:[f,g("seconds")],ss:[f,g("seconds")],m:[f,g("minutes")],mm:[f,g("minutes")],H:[f,g("hours")],h:[f,g("hours")],HH:[f,g("hours")],hh:[f,g("hours")],D:[f,g("day")],DD:[k,g("day")],Do:[_,function(p){var S=Y.ordinal,L=p.match(/\d+/);if(this.day=L[0],S)for(var F=1;F<=31;F+=1)S(F).replace(/\[|\]/g,"")===p&&(this.day=F)}],w:[f,g("week")],ww:[k,g("week")],M:[f,g("month")],MM:[k,g("month")],MMM:[_,function(p){var S=V("months"),L=(V("monthsShort")||S.map(function(F){return F.slice(0,3)})).indexOf(p)+1;if(L<1)throw new Error;this.month=L%12||L}],MMMM:[_,function(p){var S=V("months").indexOf(p)+1;if(S<1)throw new Error;this.month=S%12||S}],Y:[/[+-]?\d+/,g("year")],YY:[k,function(p){this.year=C(p)}],YYYY:[/\d{4}/,g("year")],Z:M,ZZ:M};function E(p){var S,L;S=p,L=Y&&Y.formats;for(var F=(p=S.replace(/(\[[^\]]+])|(LTS?|l{1,4}|L{1,4})/g,function(x,b,m){var w=m&&m.toUpperCase();return b||L[m]||n[m]||L[w].replace(/(\[[^\]]+])|(MMMM|MM|DD|dddd)/g,function(o,l,h){return l||h.slice(1)})})).match(a),H=F.length,X=0;X<H;X+=1){var Q=F[X],q=B[Q],y=q&&q[0],T=q&&q[1];F[X]=T?{regex:y,parser:T}:Q.replace(/^\[|\]$/g,"")}return function(x){for(var b={},m=0,w=0;m<H;m+=1){var o=F[m];if(typeof o=="string")w+=o.length;else{var l=o.regex,h=o.parser,d=x.slice(w),v=l.exec(d)[0];h.call(b,v),x=x.replace(v,"")}}return function(s){var u=s.afternoon;if(u!==void 0){var t=s.hours;u?t<12&&(s.hours+=12):t===12&&(s.hours=0),delete s.afternoon}}(b),b}}return function(p,S,L){L.p.customParseFormat=!0,p&&p.parseTwoDigitYear&&(C=p.parseTwoDigitYear);var F=S.prototype,H=F.parse;F.parse=function(X){var Q=X.date,q=X.utc,y=X.args;this.$u=q;var T=y[1];if(typeof T=="string"){var x=y[2]===!0,b=y[3]===!0,m=x||b,w=y[2];b&&(w=y[2]),Y=this.$locale(),!x&&w&&(Y=L.Ls[w]),this.$d=function(d,v,s,u){try{if(["x","X"].indexOf(v)>-1)return new Date((v==="X"?1e3:1)*d);var t=E(v)(d),I=t.year,D=t.month,A=t.day,N=t.hours,W=t.minutes,O=t.seconds,K=t.milliseconds,ae=t.zone,ie=t.week,de=new Date,fe=A||(I||D?1:de.getDate()),oe=I||de.getFullYear(),z=0;I&&!D||(z=D>0?D-1:de.getMonth());var Z,G=N||0,re=W||0,J=O||0,se=K||0;return ae?new Date(Date.UTC(oe,z,fe,G,re,J,se+60*ae.offset*1e3)):s?new Date(Date.UTC(oe,z,fe,G,re,J,se)):(Z=new Date(oe,z,fe,G,re,J,se),ie&&(Z=u(Z).week(ie).toDate()),Z)}catch{return new Date("")}}(Q,T,q,L),this.init(),w&&w!==!0&&(this.$L=this.locale(w).$L),m&&Q!=this.format(T)&&(this.$d=new Date("")),Y={}}else if(T instanceof Array)for(var o=T.length,l=1;l<=o;l+=1){y[1]=T[l-1];var h=L.apply(this,y);if(h.isValid()){this.$d=h.$d,this.$L=h.$L,this.init();break}l===o&&(this.$d=new Date(""))}else H.call(this,X)}}})})(Je);var Wt=Je.exports;const Ot=Ce(Wt);var $e={exports:{}};(function(e,r){(function(n,a){e.exports=a()})(Me,function(){var n="day";return function(a,i,k){var f=function(C){return C.add(4-C.isoWeekday(),n)},_=i.prototype;_.isoWeekYear=function(){return f(this).year()},_.isoWeek=function(C){if(!this.$utils().u(C))return this.add(7*(C-this.isoWeek()),n);var g,M,V,P,B=f(this),E=(g=this.isoWeekYear(),M=this.$u,V=(M?k.utc:k)().year(g).startOf("year"),P=4-V.isoWeekday(),V.isoWeekday()>4&&(P+=7),V.add(P,n));return B.diff(E,"week")+1},_.isoWeekday=function(C){return this.$utils().u(C)?this.day()||7:this.day(this.day()%7?C:C-7)};var Y=_.startOf;_.startOf=function(C,g){var M=this.$utils(),V=!!M.u(g)||g;return M.p(C)==="isoweek"?V?this.date(this.date()-(this.isoWeekday()-1)).startOf("day"):this.date(this.date()-1-(this.isoWeekday()-1)+7).endOf("day"):Y.bind(this)(C,g)}}})})($e);var Vt=$e.exports;const Pt=Ce(Vt);var _e=function(){var e=c(function(w,o,l,h){for(l=l||{},h=w.length;h--;l[w[h]]=o);return l},"o"),r=[6,8,10,12,13,14,15,16,17,18,20,21,22,23,24,25,26,27,28,29,30,31,33,35,36,38,40],n=[1,26],a=[1,27],i=[1,28],k=[1,29],f=[1,30],_=[1,31],Y=[1,32],C=[1,33],g=[1,34],M=[1,9],V=[1,10],P=[1,11],B=[1,12],E=[1,13],p=[1,14],S=[1,15],L=[1,16],F=[1,19],H=[1,20],X=[1,21],Q=[1,22],q=[1,23],y=[1,25],T=[1,35],x={trace:c(function(){},"trace"),yy:{},symbols_:{error:2,start:3,gantt:4,document:5,EOF:6,line:7,SPACE:8,statement:9,NL:10,weekday:11,weekday_monday:12,weekday_tuesday:13,weekday_wednesday:14,weekday_thursday:15,weekday_friday:16,weekday_saturday:17,weekday_sunday:18,weekend:19,weekend_friday:20,weekend_saturday:21,dateFormat:22,inclusiveEndDates:23,topAxis:24,axisFormat:25,tickInterval:26,excludes:27,includes:28,todayMarker:29,title:30,acc_title:31,acc_title_value:32,acc_descr:33,acc_descr_value:34,acc_descr_multiline_value:35,section:36,clickStatement:37,taskTxt:38,taskData:39,click:40,callbackname:41,callbackargs:42,href:43,clickStatementDebug:44,$accept:0,$end:1},terminals_:{2:"error",4:"gantt",6:"EOF",8:"SPACE",10:"NL",12:"weekday_monday",13:"weekday_tuesday",14:"weekday_wednesday",15:"weekday_thursday",16:"weekday_friday",17:"weekday_saturday",18:"weekday_sunday",20:"weekend_friday",21:"weekend_saturday",22:"dateFormat",23:"inclusiveEndDates",24:"topAxis",25:"axisFormat",26:"tickInterval",27:"excludes",28:"includes",29:"todayMarker",30:"title",31:"acc_title",32:"acc_title_value",33:"acc_descr",34:"acc_descr_value",35:"acc_descr_multiline_value",36:"section",38:"taskTxt",39:"taskData",40:"click",41:"callbackname",42:"callbackargs",43:"href"},productions_:[0,[3,3],[5,0],[5,2],[7,2],[7,1],[7,1],[7,1],[11,1],[11,1],[11,1],[11,1],[11,1],[11,1],[11,1],[19,1],[19,1],[9,1],[9,1],[9,1],[9,1],[9,1],[9,1],[9,1],[9,1],[9,1],[9,1],[9,1],[9,2],[9,2],[9,1],[9,1],[9,1],[9,2],[37,2],[37,3],[37,3],[37,4],[37,3],[37,4],[37,2],[44,2],[44,3],[44,3],[44,4],[44,3],[44,4],[44,2]],performAction:c(function(o,l,h,d,v,s,u){var t=s.length-1;switch(v){case 1:return s[t-1];case 2:this.$=[];break;case 3:s[t-1].push(s[t]),this.$=s[t-1];break;case 4:case 5:this.$=s[t];break;case 6:case 7:this.$=[];break;case 8:d.setWeekday("monday");break;case 9:d.setWeekday("tuesday");break;case 10:d.setWeekday("wednesday");break;case 11:d.setWeekday("thursday");break;case 12:d.setWeekday("friday");break;case 13:d.setWeekday("saturday");break;case 14:d.setWeekday("sunday");break;case 15:d.setWeekend("friday");break;case 16:d.setWeekend("saturday");break;case 17:d.setDateFormat(s[t].substr(11)),this.$=s[t].substr(11);break;case 18:d.enableInclusiveEndDates(),this.$=s[t].substr(18);break;case 19:d.TopAxis(),this.$=s[t].substr(8);break;case 20:d.setAxisFormat(s[t].substr(11)),this.$=s[t].substr(11);break;case 21:d.setTickInterval(s[t].substr(13)),this.$=s[t].substr(13);break;case 22:d.setExcludes(s[t].substr(9)),this.$=s[t].substr(9);break;case 23:d.setIncludes(s[t].substr(9)),this.$=s[t].substr(9);break;case 24:d.setTodayMarker(s[t].substr(12)),this.$=s[t].substr(12);break;case 27:d.setDiagramTitle(s[t].substr(6)),this.$=s[t].substr(6);break;case 28:this.$=s[t].trim(),d.setAccTitle(this.$);break;case 29:case 30:this.$=s[t].trim(),d.setAccDescription(this.$);break;case 31:d.addSection(s[t].substr(8)),this.$=s[t].substr(8);break;case 33:d.addTask(s[t-1],s[t]),this.$="task";break;case 34:this.$=s[t-1],d.setClickEvent(s[t-1],s[t],null);break;case 35:this.$=s[t-2],d.setClickEvent(s[t-2],s[t-1],s[t]);break;case 36:this.$=s[t-2],d.setClickEvent(s[t-2],s[t-1],null),d.setLink(s[t-2],s[t]);break;case 37:this.$=s[t-3],d.setClickEvent(s[t-3],s[t-2],s[t-1]),d.setLink(s[t-3],s[t]);break;case 38:this.$=s[t-2],d.setClickEvent(s[t-2],s[t],null),d.setLink(s[t-2],s[t-1]);break;case 39:this.$=s[t-3],d.setClickEvent(s[t-3],s[t-1],s[t]),d.setLink(s[t-3],s[t-2]);break;case 40:this.$=s[t-1],d.setLink(s[t-1],s[t]);break;case 41:case 47:this.$=s[t-1]+" "+s[t];break;case 42:case 43:case 45:this.$=s[t-2]+" "+s[t-1]+" "+s[t];break;case 44:case 46:this.$=s[t-3]+" "+s[t-2]+" "+s[t-1]+" "+s[t];break}},"anonymous"),table:[{3:1,4:[1,2]},{1:[3]},e(r,[2,2],{5:3}),{6:[1,4],7:5,8:[1,6],9:7,10:[1,8],11:17,12:n,13:a,14:i,15:k,16:f,17:_,18:Y,19:18,20:C,21:g,22:M,23:V,24:P,25:B,26:E,27:p,28:S,29:L,30:F,31:H,33:X,35:Q,36:q,37:24,38:y,40:T},e(r,[2,7],{1:[2,1]}),e(r,[2,3]),{9:36,11:17,12:n,13:a,14:i,15:k,16:f,17:_,18:Y,19:18,20:C,21:g,22:M,23:V,24:P,25:B,26:E,27:p,28:S,29:L,30:F,31:H,33:X,35:Q,36:q,37:24,38:y,40:T},e(r,[2,5]),e(r,[2,6]),e(r,[2,17]),e(r,[2,18]),e(r,[2,19]),e(r,[2,20]),e(r,[2,21]),e(r,[2,22]),e(r,[2,23]),e(r,[2,24]),e(r,[2,25]),e(r,[2,26]),e(r,[2,27]),{32:[1,37]},{34:[1,38]},e(r,[2,30]),e(r,[2,31]),e(r,[2,32]),{39:[1,39]},e(r,[2,8]),e(r,[2,9]),e(r,[2,10]),e(r,[2,11]),e(r,[2,12]),e(r,[2,13]),e(r,[2,14]),e(r,[2,15]),e(r,[2,16]),{41:[1,40],43:[1,41]},e(r,[2,4]),e(r,[2,28]),e(r,[2,29]),e(r,[2,33]),e(r,[2,34],{42:[1,42],43:[1,43]}),e(r,[2,40],{41:[1,44]}),e(r,[2,35],{43:[1,45]}),e(r,[2,36]),e(r,[2,38],{42:[1,46]}),e(r,[2,37]),e(r,[2,39])],defaultActions:{},parseError:c(function(o,l){if(l.recoverable)this.trace(o);else{var h=new Error(o);throw h.hash=l,h}},"parseError"),parse:c(function(o){var l=this,h=[0],d=[],v=[null],s=[],u=this.table,t="",I=0,D=0,A=2,N=1,W=s.slice.call(arguments,1),O=Object.create(this.lexer),K={yy:{}};for(var ae in this.yy)Object.prototype.hasOwnProperty.call(this.yy,ae)&&(K.yy[ae]=this.yy[ae]);O.setInput(o,K.yy),K.yy.lexer=O,K.yy.parser=this,typeof O.yylloc>"u"&&(O.yylloc={});var ie=O.yylloc;s.push(ie);var de=O.options&&O.options.ranges;typeof K.yy.parseError=="function"?this.parseError=K.yy.parseError:this.parseError=Object.getPrototypeOf(this).parseError;function fe(U){h.length=h.length-2*U,v.length=v.length-U,s.length=s.length-U}c(fe,"popStack");function oe(){var U;return U=d.pop()||O.lex()||N,typeof U!="number"&&(U instanceof Array&&(d=U,U=d.pop()),U=l.symbols_[U]||U),U}c(oe,"lex");for(var z,Z,G,re,J={},se,$,Re,ye;;){if(Z=h[h.length-1],this.defaultActions[Z]?G=this.defaultActions[Z]:((z===null||typeof z>"u")&&(z=oe()),G=u[Z]&&u[Z][z]),typeof G>"u"||!G.length||!G[0]){var we="";ye=[];for(se in u[Z])this.terminals_[se]&&se>A&&ye.push("'"+this.terminals_[se]+"'");O.showPosition?we="Parse error on line "+(I+1)+`:
`+O.showPosition()+`
Expecting `+ye.join(", ")+", got '"+(this.terminals_[z]||z)+"'":we="Parse error on line "+(I+1)+": Unexpected "+(z==N?"end of input":"'"+(this.terminals_[z]||z)+"'"),this.parseError(we,{text:O.match,token:this.terminals_[z]||z,line:O.yylineno,loc:ie,expected:ye})}if(G[0]instanceof Array&&G.length>1)throw new Error("Parse Error: multiple actions possible at state: "+Z+", token: "+z);switch(G[0]){case 1:h.push(z),v.push(O.yytext),s.push(O.yylloc),h.push(G[1]),z=null,D=O.yyleng,t=O.yytext,I=O.yylineno,ie=O.yylloc;break;case 2:if($=this.productions_[G[1]][1],J.$=v[v.length-$],J._$={first_line:s[s.length-($||1)].first_line,last_line:s[s.length-1].last_line,first_column:s[s.length-($||1)].first_column,last_column:s[s.length-1].last_column},de&&(J._$.range=[s[s.length-($||1)].range[0],s[s.length-1].range[1]]),re=this.performAction.apply(J,[t,D,I,K.yy,G[1],v,s].concat(W)),typeof re<"u")return re;$&&(h=h.slice(0,-1*$*2),v=v.slice(0,-1*$),s=s.slice(0,-1*$)),h.push(this.productions_[G[1]][0]),v.push(J.$),s.push(J._$),Re=u[h[h.length-2]][h[h.length-1]],h.push(Re);break;case 3:return!0}}return!0},"parse")},b=function(){var w={EOF:1,parseError:c(function(l,h){if(this.yy.parser)this.yy.parser.parseError(l,h);else throw new Error(l)},"parseError"),setInput:c(function(o,l){return this.yy=l||this.yy||{},this._input=o,this._more=this._backtrack=this.done=!1,this.yylineno=this.yyleng=0,this.yytext=this.matched=this.match="",this.conditionStack=["INITIAL"],this.yylloc={first_line:1,first_column:0,last_line:1,last_column:0},this.options.ranges&&(this.yylloc.range=[0,0]),this.offset=0,this},"setInput"),input:c(function(){var o=this._input[0];this.yytext+=o,this.yyleng++,this.offset++,this.match+=o,this.matched+=o;var l=o.match(/(?:\r\n?|\n).*/g);return l?(this.yylineno++,this.yylloc.last_line++):this.yylloc.last_column++,this.options.ranges&&this.yylloc.range[1]++,this._input=this._input.slice(1),o},"input"),unput:c(function(o){var l=o.length,h=o.split(/(?:\r\n?|\n)/g);this._input=o+this._input,this.yytext=this.yytext.substr(0,this.yytext.length-l),this.offset-=l;var d=this.match.split(/(?:\r\n?|\n)/g);this.match=this.match.substr(0,this.match.length-1),this.matched=this.matched.substr(0,this.matched.length-1),h.length-1&&(this.yylineno-=h.length-1);var v=this.yylloc.range;return this.yylloc={first_line:this.yylloc.first_line,last_line:this.yylineno+1,first_column:this.yylloc.first_column,last_column:h?(h.length===d.length?this.yylloc.first_column:0)+d[d.length-h.length].length-h[0].length:this.yylloc.first_column-l},this.options.ranges&&(this.yylloc.range=[v[0],v[0]+this.yyleng-l]),this.yyleng=this.yytext.length,this},"unput"),more:c(function(){return this._more=!0,this},"more"),reject:c(function(){if(this.options.backtrack_lexer)this._backtrack=!0;else return this.parseError("Lexical error on line "+(this.yylineno+1)+`. You can only invoke reject() in the lexer when the lexer is of the backtracking persuasion (options.backtrack_lexer = true).
`+this.showPosition(),{text:"",token:null,line:this.yylineno});return this},"reject"),less:c(function(o){this.unput(this.match.slice(o))},"less"),pastInput:c(function(){var o=this.matched.substr(0,this.matched.length-this.match.length);return(o.length>20?"...":"")+o.substr(-20).replace(/\n/g,"")},"pastInput"),upcomingInput:c(function(){var o=this.match;return o.length<20&&(o+=this._input.substr(0,20-o.length)),(o.substr(0,20)+(o.length>20?"...":"")).replace(/\n/g,"")},"upcomingInput"),showPosition:c(function(){var o=this.pastInput(),l=new Array(o.length+1).join("-");return o+this.upcomingInput()+`
`+l+"^"},"showPosition"),test_match:c(function(o,l){var h,d,v;if(this.options.backtrack_lexer&&(v={yylineno:this.yylineno,yylloc:{first_line:this.yylloc.first_line,last_line:this.last_line,first_column:this.yylloc.first_column,last_column:this.yylloc.last_column},yytext:this.yytext,match:this.match,matches:this.matches,matched:this.matched,yyleng:this.yyleng,offset:this.offset,_more:this._more,_input:this._input,yy:this.yy,conditionStack:this.conditionStack.slice(0),done:this.done},this.options.ranges&&(v.yylloc.range=this.yylloc.range.slice(0))),d=o[0].match(/(?:\r\n?|\n).*/g),d&&(this.yylineno+=d.length),this.yylloc={first_line:this.yylloc.last_line,last_line:this.yylineno+1,first_column:this.yylloc.last_column,last_column:d?d[d.length-1].length-d[d.length-1].match(/\r?\n?/)[0].length:this.yylloc.last_column+o[0].length},this.yytext+=o[0],this.match+=o[0],this.matches=o,this.yyleng=this.yytext.length,this.options.ranges&&(this.yylloc.range=[this.offset,this.offset+=this.yyleng]),this._more=!1,this._backtrack=!1,this._input=this._input.slice(o[0].length),this.matched+=o[0],h=this.performAction.call(this,this.yy,this,l,this.conditionStack[this.conditionStack.length-1]),this.done&&this._input&&(this.done=!1),h)return h;if(this._backtrack){for(var s in v)this[s]=v[s];return!1}return!1},"test_match"),next:c(function(){if(this.done)return this.EOF;this._input||(this.done=!0);var o,l,h,d;this._more||(this.yytext="",this.match="");for(var v=this._currentRules(),s=0;s<v.length;s++)if(h=this._input.match(this.rules[v[s]]),h&&(!l||h[0].length>l[0].length)){if(l=h,d=s,this.options.backtrack_lexer){if(o=this.test_match(h,v[s]),o!==!1)return o;if(this._backtrack){l=!1;continue}else return!1}else if(!this.options.flex)break}return l?(o=this.test_match(l,v[d]),o!==!1?o:!1):this._input===""?this.EOF:this.parseError("Lexical error on line "+(this.yylineno+1)+`. Unrecognized text.
`+this.showPosition(),{text:"",token:null,line:this.yylineno})},"next"),lex:c(function(){var l=this.next();return l||this.lex()},"lex"),begin:c(function(l){this.conditionStack.push(l)},"begin"),popState:c(function(){var l=this.conditionStack.length-1;return l>0?this.conditionStack.pop():this.conditionStack[0]},"popState"),_currentRules:c(function(){return this.conditionStack.length&&this.conditionStack[this.conditionStack.length-1]?this.conditions[this.conditionStack[this.conditionStack.length-1]].rules:this.conditions.INITIAL.rules},"_currentRules"),topState:c(function(l){return l=this.conditionStack.length-1-Math.abs(l||0),l>=0?this.conditionStack[l]:"INITIAL"},"topState"),pushState:c(function(l){this.begin(l)},"pushState"),stateStackSize:c(function(){return this.conditionStack.length},"stateStackSize"),options:{"case-insensitive":!0},performAction:c(function(l,h,d,v){switch(d){case 0:return this.begin("open_directive"),"open_directive";case 1:return this.begin("acc_title"),31;case 2:return this.popState(),"acc_title_value";case 3:return this.begin("acc_descr"),33;case 4:return this.popState(),"acc_descr_value";case 5:this.begin("acc_descr_multiline");break;case 6:this.popState();break;case 7:return"acc_descr_multiline_value";case 8:break;case 9:break;case 10:break;case 11:return 10;case 12:break;case 13:break;case 14:this.begin("href");break;case 15:this.popState();break;case 16:return 43;case 17:this.begin("callbackname");break;case 18:this.popState();break;case 19:this.popState(),this.begin("callbackargs");break;case 20:return 41;case 21:this.popState();break;case 22:return 42;case 23:this.begin("click");break;case 24:this.popState();break;case 25:return 40;case 26:return 4;case 27:return 22;case 28:return 23;case 29:return 24;case 30:return 25;case 31:return 26;case 32:return 28;case 33:return 27;case 34:return 29;case 35:return 12;case 36:return 13;case 37:return 14;case 38:return 15;case 39:return 16;case 40:return 17;case 41:return 18;case 42:return 20;case 43:return 21;case 44:return"date";case 45:return 30;case 46:return"accDescription";case 47:return 36;case 48:return 38;case 49:return 39;case 50:return":";case 51:return 6;case 52:return"INVALID"}},"anonymous"),rules:[/^(?:%%\{)/i,/^(?:accTitle\s*:\s*)/i,/^(?:(?!\n||)*[^\n]*)/i,/^(?:accDescr\s*:\s*)/i,/^(?:(?!\n||)*[^\n]*)/i,/^(?:accDescr\s*\{\s*)/i,/^(?:[\}])/i,/^(?:[^\}]*)/i,/^(?:%%(?!\{)*[^\n]*)/i,/^(?:[^\}]%%*[^\n]*)/i,/^(?:%%*[^\n]*[\n]*)/i,/^(?:[\n]+)/i,/^(?:\s+)/i,/^(?:%[^\n]*)/i,/^(?:href[\s]+["])/i,/^(?:["])/i,/^(?:[^"]*)/i,/^(?:call[\s]+)/i,/^(?:\([\s]*\))/i,/^(?:\()/i,/^(?:[^(]*)/i,/^(?:\))/i,/^(?:[^)]*)/i,/^(?:click[\s]+)/i,/^(?:[\s\n])/i,/^(?:[^\s\n]*)/i,/^(?:gantt\b)/i,/^(?:dateFormat\s[^#\n;]+)/i,/^(?:inclusiveEndDates\b)/i,/^(?:topAxis\b)/i,/^(?:axisFormat\s[^#\n;]+)/i,/^(?:tickInterval\s[^#\n;]+)/i,/^(?:includes\s[^#\n;]+)/i,/^(?:excludes\s[^#\n;]+)/i,/^(?:todayMarker\s[^\n;]+)/i,/^(?:weekday\s+monday\b)/i,/^(?:weekday\s+tuesday\b)/i,/^(?:weekday\s+wednesday\b)/i,/^(?:weekday\s+thursday\b)/i,/^(?:weekday\s+friday\b)/i,/^(?:weekday\s+saturday\b)/i,/^(?:weekday\s+sunday\b)/i,/^(?:weekend\s+friday\b)/i,/^(?:weekend\s+saturday\b)/i,/^(?:\d\d\d\d-\d\d-\d\d\b)/i,/^(?:title\s[^\n]+)/i,/^(?:accDescription\s[^#\n;]+)/i,/^(?:section\s[^\n]+)/i,/^(?:[^:\n]+)/i,/^(?::[^#\n;]+)/i,/^(?::)/i,/^(?:$)/i,/^(?:.)/i],conditions:{acc_descr_multiline:{rules:[6,7],inclusive:!1},acc_descr:{rules:[4],inclusive:!1},acc_title:{rules:[2],inclusive:!1},callbackargs:{rules:[21,22],inclusive:!1},callbackname:{rules:[18,19,20],inclusive:!1},href:{rules:[15,16],inclusive:!1},click:{rules:[24,25],inclusive:!1},INITIAL:{rules:[0,1,3,5,8,9,10,11,12,13,14,17,23,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52],inclusive:!0}}};return w}();x.lexer=b;function m(){this.yy={}}return c(m,"Parser"),m.prototype=x,x.Parser=m,new m}();_e.parser=_e;var zt=_e;j.extend(Pt);j.extend(Ot);j.extend(Yt);var Ue={friday:5,saturday:6},ee="",Ie="",Ae=void 0,Le="",he=[],ke=[],Fe=new Map,Ye=[],xe=[],ue="",We="",et=["active","done","crit","milestone"],Oe=[],me=!1,Ve=!1,Pe="sunday",be="saturday",De=0,Rt=c(function(){Ye=[],xe=[],ue="",Oe=[],pe=0,Ee=void 0,ve=void 0,R=[],ee="",Ie="",We="",Ae=void 0,Le="",he=[],ke=[],me=!1,Ve=!1,De=0,Fe=new Map,kt(),Pe="sunday",be="saturday"},"clear"),Nt=c(function(e){Ie=e},"setAxisFormat"),Bt=c(function(){return Ie},"getAxisFormat"),Gt=c(function(e){Ae=e},"setTickInterval"),Ht=c(function(){return Ae},"getTickInterval"),Xt=c(function(e){Le=e},"setTodayMarker"),jt=c(function(){return Le},"getTodayMarker"),qt=c(function(e){ee=e},"setDateFormat"),Ut=c(function(){me=!0},"enableInclusiveEndDates"),Zt=c(function(){return me},"endDatesAreInclusive"),Qt=c(function(){Ve=!0},"enableTopAxis"),Kt=c(function(){return Ve},"topAxisEnabled"),Jt=c(function(e){We=e},"setDisplayMode"),$t=c(function(){return We},"getDisplayMode"),es=c(function(){return ee},"getDateFormat"),ts=c(function(e){he=e.toLowerCase().split(/[\s,]+/)},"setIncludes"),ss=c(function(){return he},"getIncludes"),rs=c(function(e){ke=e.toLowerCase().split(/[\s,]+/)},"setExcludes"),ns=c(function(){return ke},"getExcludes"),as=c(function(){return Fe},"getLinks"),is=c(function(e){ue=e,Ye.push(e)},"addSection"),os=c(function(){return Ye},"getSections"),cs=c(function(){let e=Ze();const r=10;let n=0;for(;!e&&n<r;)e=Ze(),n++;return xe=R,xe},"getTasks"),tt=c(function(e,r,n,a){return a.includes(e.format(r.trim()))?!1:n.includes("weekends")&&(e.isoWeekday()===Ue[be]||e.isoWeekday()===Ue[be]+1)||n.includes(e.format("dddd").toLowerCase())?!0:n.includes(e.format(r.trim()))},"isInvalidDate"),ls=c(function(e){Pe=e},"setWeekday"),us=c(function(){return Pe},"getWeekday"),ds=c(function(e){be=e},"setWeekend"),st=c(function(e,r,n,a){if(!n.length||e.manualEndTime)return;let i;e.startTime instanceof Date?i=j(e.startTime):i=j(e.startTime,r,!0),i=i.add(1,"d");let k;e.endTime instanceof Date?k=j(e.endTime):k=j(e.endTime,r,!0);const[f,_]=fs(i,k,r,n,a);e.endTime=f.toDate(),e.renderEndTime=_},"checkTaskDates"),fs=c(function(e,r,n,a,i){let k=!1,f=null;for(;e<=r;)k||(f=r.toDate()),k=tt(e,n,a,i),k&&(r=r.add(1,"d")),e=e.add(1,"d");return[r,f]},"fixTaskDates"),Se=c(function(e,r,n){n=n.trim();const i=/^after\s+(?<ids>[\d\w- ]+)/.exec(n);if(i!==null){let f=null;for(const Y of i.groups.ids.split(" ")){let C=ne(Y);C!==void 0&&(!f||C.endTime>f.endTime)&&(f=C)}if(f)return f.endTime;const _=new Date;return _.setHours(0,0,0,0),_}let k=j(n,r.trim(),!0);if(k.isValid())return k.toDate();{Te.debug("Invalid date:"+n),Te.debug("With date format:"+r.trim());const f=new Date(n);if(f===void 0||isNaN(f.getTime())||f.getFullYear()<-1e4||f.getFullYear()>1e4)throw new Error("Invalid date:"+n);return f}},"getStartDate"),rt=c(function(e){const r=/^(\d+(?:\.\d+)?)([Mdhmswy]|ms)$/.exec(e.trim());return r!==null?[Number.parseFloat(r[1]),r[2]]:[NaN,"ms"]},"parseDuration"),nt=c(function(e,r,n,a=!1){n=n.trim();const k=/^until\s+(?<ids>[\d\w- ]+)/.exec(n);if(k!==null){let g=null;for(const V of k.groups.ids.split(" ")){let P=ne(V);P!==void 0&&(!g||P.startTime<g.startTime)&&(g=P)}if(g)return g.startTime;const M=new Date;return M.setHours(0,0,0,0),M}let f=j(n,r.trim(),!0);if(f.isValid())return a&&(f=f.add(1,"d")),f.toDate();let _=j(e);const[Y,C]=rt(n);if(!Number.isNaN(Y)){const g=_.add(Y,C);g.isValid()&&(_=g)}return _.toDate()},"getEndDate"),pe=0,le=c(function(e){return e===void 0?(pe=pe+1,"task"+pe):e},"parseId"),hs=c(function(e,r){let n;r.substr(0,1)===":"?n=r.substr(1,r.length):n=r;const a=n.split(","),i={};ze(a,i,et);for(let f=0;f<a.length;f++)a[f]=a[f].trim();let k="";switch(a.length){case 1:i.id=le(),i.startTime=e.endTime,k=a[0];break;case 2:i.id=le(),i.startTime=Se(void 0,ee,a[0]),k=a[1];break;case 3:i.id=le(a[0]),i.startTime=Se(void 0,ee,a[1]),k=a[2];break}return k&&(i.endTime=nt(i.startTime,ee,k,me),i.manualEndTime=j(k,"YYYY-MM-DD",!0).isValid(),st(i,ee,ke,he)),i},"compileData"),ks=c(function(e,r){let n;r.substr(0,1)===":"?n=r.substr(1,r.length):n=r;const a=n.split(","),i={};ze(a,i,et);for(let k=0;k<a.length;k++)a[k]=a[k].trim();switch(a.length){case 1:i.id=le(),i.startTime={type:"prevTaskEnd",id:e},i.endTime={data:a[0]};break;case 2:i.id=le(),i.startTime={type:"getStartDate",startData:a[0]},i.endTime={data:a[1]};break;case 3:i.id=le(a[0]),i.startTime={type:"getStartDate",startData:a[1]},i.endTime={data:a[2]};break}return i},"parseData"),Ee,ve,R=[],at={},ms=c(function(e,r){const n={section:ue,type:ue,processed:!1,manualEndTime:!1,renderEndTime:null,raw:{data:r},task:e,classes:[]},a=ks(ve,r);n.raw.startTime=a.startTime,n.raw.endTime=a.endTime,n.id=a.id,n.prevTaskId=ve,n.active=a.active,n.done=a.done,n.crit=a.crit,n.milestone=a.milestone,n.order=De,De++;const i=R.push(n);ve=n.id,at[n.id]=i-1},"addTask"),ne=c(function(e){const r=at[e];return R[r]},"findTaskById"),ys=c(function(e,r){const n={section:ue,type:ue,description:e,task:e,classes:[]},a=hs(Ee,r);n.startTime=a.startTime,n.endTime=a.endTime,n.id=a.id,n.active=a.active,n.done=a.done,n.crit=a.crit,n.milestone=a.milestone,Ee=n,xe.push(n)},"addTaskOrg"),Ze=c(function(){const e=c(function(n){const a=R[n];let i="";switch(R[n].raw.startTime.type){case"prevTaskEnd":{const k=ne(a.prevTaskId);a.startTime=k.endTime;break}case"getStartDate":i=Se(void 0,ee,R[n].raw.startTime.startData),i&&(R[n].startTime=i);break}return R[n].startTime&&(R[n].endTime=nt(R[n].startTime,ee,R[n].raw.endTime.data,me),R[n].endTime&&(R[n].processed=!0,R[n].manualEndTime=j(R[n].raw.endTime.data,"YYYY-MM-DD",!0).isValid(),st(R[n],ee,ke,he))),R[n].processed},"compileTask");let r=!0;for(const[n,a]of R.entries())e(n),r=r&&a.processed;return r},"compileTasks"),gs=c(function(e,r){let n=r;ce().securityLevel!=="loose"&&(n=mt(r)),e.split(",").forEach(function(a){ne(a)!==void 0&&(ot(a,()=>{window.open(n,"_self")}),Fe.set(a,n))}),it(e,"clickable")},"setLink"),it=c(function(e,r){e.split(",").forEach(function(n){let a=ne(n);a!==void 0&&a.classes.push(r)})},"setClass"),ps=c(function(e,r,n){if(ce().securityLevel!=="loose"||r===void 0)return;let a=[];if(typeof n=="string"){a=n.split(/,(?=(?:(?:[^"]*"){2})*[^"]*$)/);for(let k=0;k<a.length;k++){let f=a[k].trim();f.startsWith('"')&&f.endsWith('"')&&(f=f.substr(1,f.length-2)),a[k]=f}}a.length===0&&a.push(e),ne(e)!==void 0&&ot(e,()=>{Dt.runFunc(r,...a)})},"setClickFun"),ot=c(function(e,r){Oe.push(function(){const n=document.querySelector(`[id="${e}"]`);n!==null&&n.addEventListener("click",function(){r()})},function(){const n=document.querySelector(`[id="${e}-text"]`);n!==null&&n.addEventListener("click",function(){r()})})},"pushFun"),vs=c(function(e,r,n){e.split(",").forEach(function(a){ps(a,r,n)}),it(e,"clickable")},"setClickEvent"),Ts=c(function(e){Oe.forEach(function(r){r(e)})},"bindFunctions"),xs={getConfig:c(()=>ce().gantt,"getConfig"),clear:Rt,setDateFormat:qt,getDateFormat:es,enableInclusiveEndDates:Ut,endDatesAreInclusive:Zt,enableTopAxis:Qt,topAxisEnabled:Kt,setAxisFormat:Nt,getAxisFormat:Bt,setTickInterval:Gt,getTickInterval:Ht,setTodayMarker:Xt,getTodayMarker:jt,setAccTitle:ct,getAccTitle:lt,setDiagramTitle:ut,getDiagramTitle:dt,setDisplayMode:Jt,getDisplayMode:$t,setAccDescription:ft,getAccDescription:ht,addSection:is,getSections:os,getTasks:cs,addTask:ms,findTaskById:ne,addTaskOrg:ys,setIncludes:ts,getIncludes:ss,setExcludes:rs,getExcludes:ns,setClickEvent:vs,setLink:gs,getLinks:as,bindFunctions:Ts,parseDuration:rt,isInvalidDate:tt,setWeekday:ls,getWeekday:us,setWeekend:ds};function ze(e,r,n){let a=!0;for(;a;)a=!1,n.forEach(function(i){const k="^\\s*"+i+"\\s*$",f=new RegExp(k);e[0].match(f)&&(r[i]=!0,e.shift(1),a=!0)})}c(ze,"getTaskTags");var bs=c(function(){Te.debug("Something is calling, setConf, remove the call")},"setConf"),Qe={monday:St,tuesday:Et,wednesday:Ct,thursday:Mt,friday:It,saturday:At,sunday:Lt},ws=c((e,r)=>{let n=[...e].map(()=>-1/0),a=[...e].sort((k,f)=>k.startTime-f.startTime||k.order-f.order),i=0;for(const k of a)for(let f=0;f<n.length;f++)if(k.startTime>=n[f]){n[f]=k.endTime,k.order=f+r,f>i&&(i=f);break}return i},"getMaxIntersections"),te,_s=c(function(e,r,n,a){const i=ce().gantt,k=ce().securityLevel;let f;k==="sandbox"&&(f=ge("#i"+r));const _=k==="sandbox"?ge(f.nodes()[0].contentDocument.body):ge("body"),Y=k==="sandbox"?f.nodes()[0].contentDocument:document,C=Y.getElementById(r);te=C.parentElement.offsetWidth,te===void 0&&(te=1200),i.useWidth!==void 0&&(te=i.useWidth);const g=a.db.getTasks();let M=[];for(const y of g)M.push(y.type);M=q(M);const V={};let P=2*i.topPadding;if(a.db.getDisplayMode()==="compact"||i.displayMode==="compact"){const y={};for(const x of g)y[x.section]===void 0?y[x.section]=[x]:y[x.section].push(x);let T=0;for(const x of Object.keys(y)){const b=ws(y[x],T)+1;T+=b,P+=b*(i.barHeight+i.barGap),V[x]=b}}else{P+=g.length*(i.barHeight+i.barGap);for(const y of M)V[y]=g.filter(T=>T.type===y).length}C.setAttribute("viewBox","0 0 "+te+" "+P);const B=_.select(`[id="${r}"]`),E=yt().domain([gt(g,function(y){return y.startTime}),pt(g,function(y){return y.endTime})]).rangeRound([0,te-i.leftPadding-i.rightPadding]);function p(y,T){const x=y.startTime,b=T.startTime;let m=0;return x>b?m=1:x<b&&(m=-1),m}c(p,"taskCompare"),g.sort(p),S(g,te,P),vt(B,P,te,i.useMaxWidth),B.append("text").text(a.db.getDiagramTitle()).attr("x",te/2).attr("y",i.titleTopMargin).attr("class","titleText");function S(y,T,x){const b=i.barHeight,m=b+i.barGap,w=i.topPadding,o=i.leftPadding,l=Tt().domain([0,M.length]).range(["#00B9FA","#F95002"]).interpolate(xt);F(m,w,o,T,x,y,a.db.getExcludes(),a.db.getIncludes()),H(o,w,T,x),L(y,m,w,o,b,l,T),X(m,w),Q(o,w,T,x)}c(S,"makeGantt");function L(y,T,x,b,m,w,o){const h=[...new Set(y.map(u=>u.order))].map(u=>y.find(t=>t.order===u));B.append("g").selectAll("rect").data(h).enter().append("rect").attr("x",0).attr("y",function(u,t){return t=u.order,t*T+x-2}).attr("width",function(){return o-i.rightPadding/2}).attr("height",T).attr("class",function(u){for(const[t,I]of M.entries())if(u.type===I)return"section section"+t%i.numberSectionStyles;return"section section0"});const d=B.append("g").selectAll("rect").data(y).enter(),v=a.db.getLinks();if(d.append("rect").attr("id",function(u){return u.id}).attr("rx",3).attr("ry",3).attr("x",function(u){return u.milestone?E(u.startTime)+b+.5*(E(u.endTime)-E(u.startTime))-.5*m:E(u.startTime)+b}).attr("y",function(u,t){return t=u.order,t*T+x}).attr("width",function(u){return u.milestone?m:E(u.renderEndTime||u.endTime)-E(u.startTime)}).attr("height",m).attr("transform-origin",function(u,t){return t=u.order,(E(u.startTime)+b+.5*(E(u.endTime)-E(u.startTime))).toString()+"px "+(t*T+x+.5*m).toString()+"px"}).attr("class",function(u){const t="task";let I="";u.classes.length>0&&(I=u.classes.join(" "));let D=0;for(const[N,W]of M.entries())u.type===W&&(D=N%i.numberSectionStyles);let A="";return u.active?u.crit?A+=" activeCrit":A=" active":u.done?u.crit?A=" doneCrit":A=" done":u.crit&&(A+=" crit"),A.length===0&&(A=" task"),u.milestone&&(A=" milestone "+A),A+=D,A+=" "+I,t+A}),d.append("text").attr("id",function(u){return u.id+"-text"}).text(function(u){return u.task}).attr("font-size",i.fontSize).attr("x",function(u){let t=E(u.startTime),I=E(u.renderEndTime||u.endTime);u.milestone&&(t+=.5*(E(u.endTime)-E(u.startTime))-.5*m),u.milestone&&(I=t+m);const D=this.getBBox().width;return D>I-t?I+D+1.5*i.leftPadding>o?t+b-5:I+b+5:(I-t)/2+t+b}).attr("y",function(u,t){return t=u.order,t*T+i.barHeight/2+(i.fontSize/2-2)+x}).attr("text-height",m).attr("class",function(u){const t=E(u.startTime);let I=E(u.endTime);u.milestone&&(I=t+m);const D=this.getBBox().width;let A="";u.classes.length>0&&(A=u.classes.join(" "));let N=0;for(const[O,K]of M.entries())u.type===K&&(N=O%i.numberSectionStyles);let W="";return u.active&&(u.crit?W="activeCritText"+N:W="activeText"+N),u.done?u.crit?W=W+" doneCritText"+N:W=W+" doneText"+N:u.crit&&(W=W+" critText"+N),u.milestone&&(W+=" milestoneText"),D>I-t?I+D+1.5*i.leftPadding>o?A+" taskTextOutsideLeft taskTextOutside"+N+" "+W:A+" taskTextOutsideRight taskTextOutside"+N+" "+W+" width-"+D:A+" taskText taskText"+N+" "+W+" width-"+D}),ce().securityLevel==="sandbox"){let u;u=ge("#i"+r);const t=u.nodes()[0].contentDocument;d.filter(function(I){return v.has(I.id)}).each(function(I){var D=t.querySelector("#"+I.id),A=t.querySelector("#"+I.id+"-text");const N=D.parentNode;var W=t.createElement("a");W.setAttribute("xlink:href",v.get(I.id)),W.setAttribute("target","_top"),N.appendChild(W),W.appendChild(D),W.appendChild(A)})}}c(L,"drawRects");function F(y,T,x,b,m,w,o,l){if(o.length===0&&l.length===0)return;let h,d;for(const{startTime:D,endTime:A}of w)(h===void 0||D<h)&&(h=D),(d===void 0||A>d)&&(d=A);if(!h||!d)return;if(j(d).diff(j(h),"year")>5){Te.warn("The difference between the min and max time is more than 5 years. This will cause performance issues. Skipping drawing exclude days.");return}const v=a.db.getDateFormat(),s=[];let u=null,t=j(h);for(;t.valueOf()<=d;)a.db.isInvalidDate(t,v,o,l)?u?u.end=t:u={start:t,end:t}:u&&(s.push(u),u=null),t=t.add(1,"d");B.append("g").selectAll("rect").data(s).enter().append("rect").attr("id",function(D){return"exclude-"+D.start.format("YYYY-MM-DD")}).attr("x",function(D){return E(D.start)+x}).attr("y",i.gridLineStartPadding).attr("width",function(D){const A=D.end.add(1,"day");return E(A)-E(D.start)}).attr("height",m-T-i.gridLineStartPadding).attr("transform-origin",function(D,A){return(E(D.start)+x+.5*(E(D.end)-E(D.start))).toString()+"px "+(A*y+.5*m).toString()+"px"}).attr("class","exclude-range")}c(F,"drawExcludeDays");function H(y,T,x,b){let m=bt(E).tickSize(-b+T+i.gridLineStartPadding).tickFormat(Ne(a.db.getAxisFormat()||i.axisFormat||"%Y-%m-%d"));const o=/^([1-9]\d*)(millisecond|second|minute|hour|day|week|month)$/.exec(a.db.getTickInterval()||i.tickInterval);if(o!==null){const l=o[1],h=o[2],d=a.db.getWeekday()||i.weekday;switch(h){case"millisecond":m.ticks(qe.every(l));break;case"second":m.ticks(je.every(l));break;case"minute":m.ticks(Xe.every(l));break;case"hour":m.ticks(He.every(l));break;case"day":m.ticks(Ge.every(l));break;case"week":m.ticks(Qe[d].every(l));break;case"month":m.ticks(Be.every(l));break}}if(B.append("g").attr("class","grid").attr("transform","translate("+y+", "+(b-50)+")").call(m).selectAll("text").style("text-anchor","middle").attr("fill","#000").attr("stroke","none").attr("font-size",10).attr("dy","1em"),a.db.topAxisEnabled()||i.topAxis){let l=wt(E).tickSize(-b+T+i.gridLineStartPadding).tickFormat(Ne(a.db.getAxisFormat()||i.axisFormat||"%Y-%m-%d"));if(o!==null){const h=o[1],d=o[2],v=a.db.getWeekday()||i.weekday;switch(d){case"millisecond":l.ticks(qe.every(h));break;case"second":l.ticks(je.every(h));break;case"minute":l.ticks(Xe.every(h));break;case"hour":l.ticks(He.every(h));break;case"day":l.ticks(Ge.every(h));break;case"week":l.ticks(Qe[v].every(h));break;case"month":l.ticks(Be.every(h));break}}B.append("g").attr("class","grid").attr("transform","translate("+y+", "+T+")").call(l).selectAll("text").style("text-anchor","middle").attr("fill","#000").attr("stroke","none").attr("font-size",10)}}c(H,"makeGrid");function X(y,T){let x=0;const b=Object.keys(V).map(m=>[m,V[m]]);B.append("g").selectAll("text").data(b).enter().append(function(m){const w=m[0].split(_t.lineBreakRegex),o=-(w.length-1)/2,l=Y.createElementNS("http://www.w3.org/2000/svg","text");l.setAttribute("dy",o+"em");for(const[h,d]of w.entries()){const v=Y.createElementNS("http://www.w3.org/2000/svg","tspan");v.setAttribute("alignment-baseline","central"),v.setAttribute("x","10"),h>0&&v.setAttribute("dy","1em"),v.textContent=d,l.appendChild(v)}return l}).attr("x",10).attr("y",function(m,w){if(w>0)for(let o=0;o<w;o++)return x+=b[w-1][1],m[1]*y/2+x*y+T;else return m[1]*y/2+T}).attr("font-size",i.sectionFontSize).attr("class",function(m){for(const[w,o]of M.entries())if(m[0]===o)return"sectionTitle sectionTitle"+w%i.numberSectionStyles;return"sectionTitle"})}c(X,"vertLabels");function Q(y,T,x,b){const m=a.db.getTodayMarker();if(m==="off")return;const w=B.append("g").attr("class","today"),o=new Date,l=w.append("line");l.attr("x1",E(o)+y).attr("x2",E(o)+y).attr("y1",i.titleTopMargin).attr("y2",b-i.titleTopMargin).attr("class","today"),m!==""&&l.attr("style",m.replace(/,/g,";"))}c(Q,"drawToday");function q(y){const T={},x=[];for(let b=0,m=y.length;b<m;++b)Object.prototype.hasOwnProperty.call(T,y[b])||(T[y[b]]=!0,x.push(y[b]));return x}c(q,"checkUnique")},"draw"),Ds={setConf:bs,draw:_s},Ss=c(e=>`
  .mermaid-main-font {
        font-family: ${e.fontFamily};
  }

  .exclude-range {
    fill: ${e.excludeBkgColor};
  }

  .section {
    stroke: none;
    opacity: 0.2;
  }

  .section0 {
    fill: ${e.sectionBkgColor};
  }

  .section2 {
    fill: ${e.sectionBkgColor2};
  }

  .section1,
  .section3 {
    fill: ${e.altSectionBkgColor};
    opacity: 0.2;
  }

  .sectionTitle0 {
    fill: ${e.titleColor};
  }

  .sectionTitle1 {
    fill: ${e.titleColor};
  }

  .sectionTitle2 {
    fill: ${e.titleColor};
  }

  .sectionTitle3 {
    fill: ${e.titleColor};
  }

  .sectionTitle {
    text-anchor: start;
    font-family: ${e.fontFamily};
  }


  /* Grid and axis */

  .grid .tick {
    stroke: ${e.gridColor};
    opacity: 0.8;
    shape-rendering: crispEdges;
  }

  .grid .tick text {
    font-family: ${e.fontFamily};
    fill: ${e.textColor};
  }

  .grid path {
    stroke-width: 0;
  }


  /* Today line */

  .today {
    fill: none;
    stroke: ${e.todayLineColor};
    stroke-width: 2px;
  }


  /* Task styling */

  /* Default task */

  .task {
    stroke-width: 2;
  }

  .taskText {
    text-anchor: middle;
    font-family: ${e.fontFamily};
  }

  .taskTextOutsideRight {
    fill: ${e.taskTextDarkColor};
    text-anchor: start;
    font-family: ${e.fontFamily};
  }

  .taskTextOutsideLeft {
    fill: ${e.taskTextDarkColor};
    text-anchor: end;
  }


  /* Special case clickable */

  .task.clickable {
    cursor: pointer;
  }

  .taskText.clickable {
    cursor: pointer;
    fill: ${e.taskTextClickableColor} !important;
    font-weight: bold;
  }

  .taskTextOutsideLeft.clickable {
    cursor: pointer;
    fill: ${e.taskTextClickableColor} !important;
    font-weight: bold;
  }

  .taskTextOutsideRight.clickable {
    cursor: pointer;
    fill: ${e.taskTextClickableColor} !important;
    font-weight: bold;
  }


  /* Specific task settings for the sections*/

  .taskText0,
  .taskText1,
  .taskText2,
  .taskText3 {
    fill: ${e.taskTextColor};
  }

  .task0,
  .task1,
  .task2,
  .task3 {
    fill: ${e.taskBkgColor};
    stroke: ${e.taskBorderColor};
  }

  .taskTextOutside0,
  .taskTextOutside2
  {
    fill: ${e.taskTextOutsideColor};
  }

  .taskTextOutside1,
  .taskTextOutside3 {
    fill: ${e.taskTextOutsideColor};
  }


  /* Active task */

  .active0,
  .active1,
  .active2,
  .active3 {
    fill: ${e.activeTaskBkgColor};
    stroke: ${e.activeTaskBorderColor};
  }

  .activeText0,
  .activeText1,
  .activeText2,
  .activeText3 {
    fill: ${e.taskTextDarkColor} !important;
  }


  /* Completed task */

  .done0,
  .done1,
  .done2,
  .done3 {
    stroke: ${e.doneTaskBorderColor};
    fill: ${e.doneTaskBkgColor};
    stroke-width: 2;
  }

  .doneText0,
  .doneText1,
  .doneText2,
  .doneText3 {
    fill: ${e.taskTextDarkColor} !important;
  }


  /* Tasks on the critical line */

  .crit0,
  .crit1,
  .crit2,
  .crit3 {
    stroke: ${e.critBorderColor};
    fill: ${e.critBkgColor};
    stroke-width: 2;
  }

  .activeCrit0,
  .activeCrit1,
  .activeCrit2,
  .activeCrit3 {
    stroke: ${e.critBorderColor};
    fill: ${e.activeTaskBkgColor};
    stroke-width: 2;
  }

  .doneCrit0,
  .doneCrit1,
  .doneCrit2,
  .doneCrit3 {
    stroke: ${e.critBorderColor};
    fill: ${e.doneTaskBkgColor};
    stroke-width: 2;
    cursor: pointer;
    shape-rendering: crispEdges;
  }

  .milestone {
    transform: rotate(45deg) scale(0.8,0.8);
  }

  .milestoneText {
    font-style: italic;
  }
  .doneCritText0,
  .doneCritText1,
  .doneCritText2,
  .doneCritText3 {
    fill: ${e.taskTextDarkColor} !important;
  }

  .activeCritText0,
  .activeCritText1,
  .activeCritText2,
  .activeCritText3 {
    fill: ${e.taskTextDarkColor} !important;
  }

  .titleText {
    text-anchor: middle;
    font-size: 18px;
    fill: ${e.titleColor||e.textColor};
    font-family: ${e.fontFamily};
  }
`,"getStyles"),Es=Ss,Is={parser:zt,db:xs,renderer:Ds,styles:Es};export{Is as diagram};
