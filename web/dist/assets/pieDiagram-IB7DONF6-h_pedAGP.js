import{p as B}from"./chunk-4BMEZGHF-DFDMCTez.js";import{E as U,o as K,p as Q,s as V,g as Z,c as j,b as q,_ as i,l as C,v as H,e as J,F as X,K as Y,N as tt,O as z,P as et,m as at,Q as rt}from"./mermaid-9C29qDeS.js";import{p as it}from"./radar-MK3ICKWK-BAKYc4z2.js";import"./react-core-CZZu3mg-.js";import"./_baseUniq-CcXtgN7d.js";import"./index-B51b6nwH.js";import"./semi-ui-BhhGO3pb.js";import"./i18n-C3M_f-G6.js";import"./tools-tHJQmXK5.js";import"./react-components-DkXYjmkh.js";import"./vchart-DdCisKjc.js";import"./_basePickBy-C7093ex6.js";import"./clone-BfTPD0y4.js";var F=U.pie,D={sections:new Map,showData:!1,config:F},f=D.sections,w=D.showData,st=structuredClone(F),ot=i(()=>structuredClone(st),"getConfig"),nt=i(()=>{f=new Map,w=D.showData,H()},"clear"),lt=i(({label:t,value:a})=>{f.has(t)||(f.set(t,a),C.debug(`added new section: ${t}, with value: ${a}`))},"addSection"),ct=i(()=>f,"getSections"),pt=i(t=>{w=t},"setShowData"),dt=i(()=>w,"getShowData"),G={getConfig:ot,clear:nt,setDiagramTitle:K,getDiagramTitle:Q,setAccTitle:V,getAccTitle:Z,setAccDescription:j,getAccDescription:q,addSection:lt,getSections:ct,setShowData:pt,getShowData:dt},gt=i((t,a)=>{B(t,a),a.setShowData(t.showData),t.sections.map(a.addSection)},"populateDb"),mt={parse:i(async t=>{const a=await it("pie",t);C.debug(a),gt(a,G)},"parse")},ut=i(t=>`
  .pieCircle{
    stroke: ${t.pieStrokeColor};
    stroke-width : ${t.pieStrokeWidth};
    opacity : ${t.pieOpacity};
  }
  .pieOuterCircle{
    stroke: ${t.pieOuterStrokeColor};
    stroke-width: ${t.pieOuterStrokeWidth};
    fill: none;
  }
  .pieTitleText {
    text-anchor: middle;
    font-size: ${t.pieTitleTextSize};
    fill: ${t.pieTitleTextColor};
    font-family: ${t.fontFamily};
  }
  .slice {
    font-family: ${t.fontFamily};
    fill: ${t.pieSectionTextColor};
    font-size:${t.pieSectionTextSize};
    // fill: white;
  }
  .legend text {
    fill: ${t.pieLegendTextColor};
    font-family: ${t.fontFamily};
    font-size: ${t.pieLegendTextSize};
  }
`,"getStyles"),ft=ut,ht=i(t=>{const a=[...t.entries()].map(s=>({label:s[0],value:s[1]})).sort((s,n)=>n.value-s.value);return rt().value(s=>s.value)(a)},"createPieArcs"),vt=i((t,a,O,s)=>{C.debug(`rendering pie chart
`+t);const n=s.db,y=J(),T=X(n.getConfig(),y.pie),$=40,o=18,d=4,c=450,h=c,v=Y(a),l=v.append("g");l.attr("transform","translate("+h/2+","+c/2+")");const{themeVariables:r}=y;let[A]=tt(r.pieOuterStrokeWidth);A??(A=2);const _=T.textPosition,g=Math.min(h,c)/2-$,P=z().innerRadius(0).outerRadius(g),W=z().innerRadius(g*_).outerRadius(g*_);l.append("circle").attr("cx",0).attr("cy",0).attr("r",g+A/2).attr("class","pieOuterCircle");const E=n.getSections(),S=ht(E),M=[r.pie1,r.pie2,r.pie3,r.pie4,r.pie5,r.pie6,r.pie7,r.pie8,r.pie9,r.pie10,r.pie11,r.pie12],p=et(M);l.selectAll("mySlices").data(S).enter().append("path").attr("d",P).attr("fill",e=>p(e.data.label)).attr("class","pieCircle");let b=0;E.forEach(e=>{b+=e}),l.selectAll("mySlices").data(S).enter().append("text").text(e=>(e.data.value/b*100).toFixed(0)+"%").attr("transform",e=>"translate("+W.centroid(e)+")").style("text-anchor","middle").attr("class","slice"),l.append("text").text(n.getDiagramTitle()).attr("x",0).attr("y",-400/2).attr("class","pieTitleText");const x=l.selectAll(".legend").data(p.domain()).enter().append("g").attr("class","legend").attr("transform",(e,m)=>{const u=o+d,I=u*p.domain().length/2,L=12*o,N=m*u-I;return"translate("+L+","+N+")"});x.append("rect").attr("width",o).attr("height",o).style("fill",p).style("stroke",p),x.data(S).append("text").attr("x",o+d).attr("y",o-d).text(e=>{const{label:m,value:u}=e.data;return n.getShowData()?`${m} [${u}]`:m});const R=Math.max(...x.selectAll("text").nodes().map(e=>(e==null?void 0:e.getBoundingClientRect().width)??0)),k=h+$+o+d+R;v.attr("viewBox",`0 0 ${k} ${c}`),at(v,c,k,T.useMaxWidth)},"draw"),St={draw:vt},Ft={parser:mt,db:G,renderer:St,styles:ft};export{Ft as diagram};
