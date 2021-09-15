# Extract Sources
An extract source represents either a streaming API like a JSON feed or a type of file like CSV or Excel.  A churro pipeline lets you define multiple extract sources which it will use to extract data from and load into the pipeline backend database.

## Excel Files
An example of an extract source for an Excel file would look like the following:
```yaml
    extractrules:
    - columnname: city
      columnpath: "1"
      columntype: TEXT
      extractsourceid: c4p8hhhjiqcs73ee0lsg
      id: c4p8hkpjiqcs73ee0lt0
      matchvalues: ""
      transformfunctionname: None  
    extractsources:
    - cronexpression: 30s
      id: c4p8hhhjiqcs73ee0lsg
      name: my-xlsx-files
      path: /churro/xlsxfiles
      regex: '[a-z,0-9].(xlsx)$'
      scheme: xlsx
      skipheaders: 1
      tablename: myxlsxtable
```

With Excel files, you can specify how many header rows there are to skip during
processing.  In the above example, we specify that we want to skip a single header row, store the processed rows into the *myxlsxtable* database backend table, and also process files from the */churro/xlsxfiles* directory that have a *.xlsx* file extension.  

A single extract rule is associated with the extract source, we 
reference a column we call *city* by specifying a *path* of *1*.  For Excel
files, we define columns as starting from *0*, so in this example the *city* value
is in the second column of the spreadsheet file.

## CSV Files
Comma Separated Files (CSV) are processed by churro.  As with Excel files, we reference each CSV column value by an integer value starting with *0*.

Here is an example of a CSV extract source definition:
```yaml
    extractrules:
    - columnname: a
      columnpath: "1"
      columntype: TEXT
      extractsourceid: c4p6rg1jiqcs73d90500
      id: c4p6ri1jiqcs73d9050g
      matchvalues: ""
      transformfunctionname: None
    extractsources:
    - cronexpression: 30s
      id: c4p6rg1jiqcs73d90500
      name: my-csv-files
      path: /churro/csvfiles
      regex: '[a-z,0-9].(csv)$'
      scheme: csv
      tablename: mycsvtable
```

In the above example, we define an extract rule for a column we call *a*, that has a column path of *1*.  This value is the second CSV column value in each row.

We can tell churro to also skip header rows by specifying a *skipheaders* value.

## XML Files

Files of XML content can be processed by churro with a definition similar to
the following:
```yaml
   extractrules:
    - columnname: author
      columnpath: /library/book/author/name
      columntype: TEXT
      extractsourceid: c4qhjipjiqcs73a43460
      id: c4qhkfhjiqcs73a4346g
      matchvalues: ""
      transformfunctionname: None
    extractsources:
    - cronexpression: 30s
      id: c4qhjipjiqcs73a43460
      name: my-xml-files
      path: /churro/xmlfiles
      regex: '[a-z,0-9].(xml)$'
      scheme: xml
      tablename: myxmltable
```

Notice that we are extract a column that we name *author*.  We use an xmlpath expression to locate the value we want, e.g. */library/book/author/name*.

## JSON Path Message Files
churro can process files that contain a single JSON message, and you can extract values from the JSON message using jsonpath expressions.

Here is an example of an extract source for jsonpath processing:
```yaml
    extractrules:
    - columnname: author
      columnpath: $..book[*].author
      columntype: TEXT
      extractsourceid: c4qi3epjiqcs73a434c0
      id: c4qi3p1jiqcs73a434cg
      matchvalues: ""
      transformfunctionname: transforms.MyUppercase
    extractsources:
    - cronexpression: 30s
      id: c4qi3epjiqcs73a434c0
      name: my-jsonpath-files
      path: /churro/jsonpathfiles
      regex: '[a-z,0-9].(jsonpath)$'
      scheme: jsonpath
      tablename: myjsonpathtable
```

In this example, we are extract an *author* field from the JSON message file using a jsonpath expression of *$..book[*].author*.  We also are performing a transformation function of *MyUppercase* on the value before we insert it into the pipeline database table *myjsonpathtable*.


## Raw JSON Files
churro can ingest a raw JSON file and store it within a JSONB database column without performing any field extractions.  This type of JSON processing is meant for situations where you want to store the raw JSON directly into the database as a single database row with the JSON message stored in a single column.

In this extract source type, there is no transformation function capability offered by churro.

An example of an extract source definition for raw json files looks like:
```yaml
    extractrules:
    extractsources:
    - cronexpression: 30s
      id: c4qinf9jiqcs73a434d0
      name: my-json-files
      path: /churro/jsonfiles
      regex: '[a-z,0-9].(json)$'
      scheme: json
      tablename: myjsontable
```
Notice in this file type, that no extract rules are specified, the entire json message is extracted and loaded into the pipeline table *myjsontable*.

## JSON API Stream
churro can continually process data from a JSON API and store it in the pipeline database.  Here is an example that defines a SpaceX Starlink extract source:
```yaml
    extractrules:
    - columnname: shipname
      columnpath: $[*].spaceTrack.OBJECT_NAME
      columntype: TEXT
      extractsourceid: c4qi3epjiqcs73a434c0
      id: c4qi3p1jiqcs73a434cg
      matchvalues: ""
      transformfunctionname: transforms.MyUppercase
    - columnname: longitude
      columnpath: $[*].longitude
      columntype: DECIMAL
      extractsourceid: c4qi3epjiqcs73a434c0
      id: c4qi3p1jiqcs73a434ch
    - columnname: latitude
      columnpath: $[*].latitude
      columntype: DECIMAL
      extractsourceid: c4qi3epjiqcs73a434c0
      id: c4qi3p1jiqcs73a434ci
    extractsources:
    - cronexpression: @every 1h
      id: c4qi3epjiqcs73a434c0
      name: my-starlink-feed
      path: https://api.spacexdata.com/v4/starlink
      scheme: api
      tablename: mystarlinktable
```

This example causes the SpaceX feed to be polled every hour based on the *cronexpression*.  There are 3 extract rules defined:  shipname, latitude, longitude.  The extracted values
are stored in the *mystarlinktable* database table in the pipeline's backend database.
