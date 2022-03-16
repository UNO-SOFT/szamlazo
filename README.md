# kyoto starter
Quick Start project setup

## What's included

- [kyoto](https://github.com/kyoto-framework/kyoto)
- [kyoto uikit](https://github.com/kyoto-framework/uikit)
- [tailwindcss](https://tailwindcss.com)

## How to use

- Clone project with `git clone --recursive https://github.com/kyoto-framework/starter <app name>` (replace `<app name>` with desired app name)
- Install dependencies for statics with executing `npm i` in static folder
- Build statics with executing `npm run build` in static folder
- And execute `go run .` in the root to start the server

# Tapasztalatok
## Általános
Go html/template-et használ, ami lehet hogy nem a leggyorsabb (https://github.com/SlinSo/goTemplateBenchmark), cserébe garantáltan biztonságos !

A minden oldal (page), komponens (component) kerüljön külön fájl(párba, mert külön a .go és külön a .html)
elsőre túl aprózónak tűnik, de nagyon jól szeparálja az elemeket, és könnyen újra felhasználhatóvá teszi őket.

Sok munka, de megtérül újra felhasználható építőköveket készíteni.

## Statikus
A `template.ParseGlob` a path-t levágja (`template.ParseGlob("uikit/twui/*.html")` esetén csak `xxx.html` lesz a template neve, NEM `uikit/twui/xxx.html`),
ezért egyszerűbb a zip-et is könyvtáranként csinálni.

A `kyoto-uikit`-ben a `static/dist`-et kell használni, `static/dist` nélkül (`http.StripPrefix("/static/", ...)`).
A frissítés `cd static && npm run build` -el történik.

## Dinamikus
A teljes oldal állapotát (state) küldözgeti a hívások során, json sztringekként kódolva, tehát
1. exportáltnak kell lennie (nagybetűvel kezdődjön a mező neve),
2. sztringként vissza kell tudni olvasni json-ba, tehát kellhet külön json.Unmarshaler;
3. az állapot legyen a lehető legkevesebb.
    
Nem elfelejteni a komponens első div-jébe beletenni a `{{ componentattrs . }}`-t!

`onchange='{{ bind "Mezőneve" }}'` teszi az input értékét a struct mezőjébe,
`'{{ action "Reload" }}'` -al lehet a `Reload` action-t meghívni - amit a struct `Actions()` metódusa kell visszaadjon.

Nem teljesen értem miért (én úgy olvasom a [text/template](pkg.go.dev/text/template)-et hogy jól kellene működnie),
de a `with` felülírja a `.` értékét - a semmivel, mert nem ad vissza semmit.
A legegyszerűbb kerülő út magát a `.`-ot lementeni egy változóba `with`-el, és azt használni.
