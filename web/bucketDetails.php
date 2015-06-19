<html>
  <head>
    <style type="text/css">
      tr:nth-child(even){
        background-color:white;
      }
      tr:nth-child(odd){
        background-color:Lavender;
      }
    </style>
  </head>
<body>
<?php
$days = 1;
$bucket = " is null";

if(isset($_GET["days"]))
{
	if(is_numeric($_GET["days"]))
	{
		if($_GET["days"] > 0)
		{
			$days = $_GET["days"];
		}
	}
}

if(isset($_GET["id"]))
{
	if(is_numeric($_GET["id"]))
	{
		if($_GET["id"] > 0)
		{
			$bucket = " = " . $_GET["id"];
		}
	}
}

$requested_domain_bitfield = 0;
if(isset($_GET["domains"]))
{
        if(is_numeric($_GET["domains"]))
        {
                if($_GET["domains"] > 0)
                {
                        $requested_domain_bitfield = $_GET["domains"];
                }
        }
}

include 'database.php';

// how far back are we going to be looking for events?
// We need an actual value here so that when we use it in a where clause, that value gets pushed to the foriegn servers
$query = "select now()-interval '1 day'*$days";
$result = pg_query($query) or die('Query failed: ' . pg_last_error());
$timespan = pg_fetch_result($result,0,0);

// calculate the bitfield for everything
$query = "select bit_or((2^(id-1))::int) from data_sources";
$result = pg_query($query) or die('query failed: ' . pg_last_error());
$all_domains = pg_fetch_result($result,0,0);

// if we don't specify any particular domain, just assume we mean all of them.
if($requested_domain_bitfield == 0)
{
	$requested_domain_bitfield = $all_domains;
}

// decompose our domain bitfield to see which data sources we actually want to query
$selected_datasources = array();
$query = "with s as (select 2^(id-1) as bits,schema,name from data_sources order by id) select schema,name from s where bits::int & $requested_domain_bitfield > 0";
$result = pg_query($query) or die('Query failed: ' . pg_last_error());
while($ds = pg_fetch_row($result))
{
        $selected_datasources[] = array($ds[0],$ds[1]);
}
$events = "select finished,host,event,'" . $selected_datasources[0][1] . "' as domain from \"" . $selected_datasources[0][0] . "\".events where finished > '$timespan' and bucket_id$bucket";
for($i=1;$i < count($selected_datasources); $i++)
{
        $events = $events . " union all select finished,host,event,'" . $selected_datasources[$i][1] . "' as domain from \"" . $selected_datasources[$i][0] . "\".events where finished > '$timespan' and bucket_id$bucket";
}


$query = "with all_events as ($events) SELECT finished,host,event,domain from all_events order by finished desc limit 5000";
if (isset($_GET["n"]))
{
	$query = "with all_events as ($events) SELECT count(*),max(finished),normalize_query(event) from all_events group by normalize_query(event) order by count(*)  desc limit 5000";
}

$events = pg_query($query) or die('Query failed: ' . pg_last_error());

echo "<h2>Showing the " . pg_num_rows($events) . " instances of ";

if ($bucket == " is null")
{
	echo "interesting things";
}
else
{
	$query = "SELECT name from \"" . $selected_datasources[0][0] . "\".buckets where id $bucket";
	$bucket_info = pg_query($query) or die('Query failed: ' . pg_last_error());
	while ($row = pg_fetch_row($bucket_info))
	{
		echo $row[0];
	}
	pg_free_result($bucket_info);
}

function second_value($a)
{
	return $a[1];
}

$pretty_datasources = "everywhere";
if($requested_domain_bitfield > 0 && $requested_domain_bitfield != $all_domains)
{
        $pretty_datasources = implode(", ", array_map("second_value",$selected_datasources));
}
echo " over the last $days days from $pretty_datasources</h2>";

echo "(back to the <a href='overview.php?days=$days&domains=$requested_domain_bitfield'>overview)<p>";

$me = "<a href='bucketDetails.php?days=$days&domains=$requested_domain_bitfield";
if ($bucket != " is null")
{
	$me = $me . "&id=" . $_GET["id"];
}

if (isset($_GET["n"]))
{
	echo "[ $me'>Show queries as they were</a> | Showing queries normalized ]";
}
else
{
	echo "[ Showing queries as they were | $me&n=1'>Show queries normalized</a> ]";
}

echo "<hr><table>\n";
    echo "\t<tr>\n";
if (isset($_GET["n"]))
{
        echo "\t\t<td>Count</td>\n";
        echo "\t\t<td>Most Recent</td>\n";
        echo "\t\t<td>What</td>\n";
    echo "\t</tr>\n";
  while ($row = pg_fetch_row($events))
  {
    echo "\t<tr>\n";
	echo "\t\t<td>" . htmlentities($row[0], ENT_QUOTES) . "</td>\n";
	echo "\t\t<td>" . htmlentities($row[1], ENT_QUOTES) . "</td>\n";
	echo "\t\t<td><pre>" . $row[2] . "</pre></td>\n";
    echo "\t</tr>\n";
  }
}
else
{
        echo "\t\t<td>When</td>\n";
        echo "\t\t<td>Domain</td>\n";
        echo "\t\t<td>Host</td>\n";
        echo "\t\t<td>What</td>\n";
    echo "\t</tr>\n";
}
while ($row = pg_fetch_row($events))
{
    echo "\t<tr>\n";
	echo "\t\t<td>" . htmlentities($row[0], ENT_QUOTES) . "</td>\n";
	echo "\t\t<td>" . htmlentities($row[3], ENT_QUOTES) . "</td>\n";
	echo "\t\t<td>" . htmlentities($row[1], ENT_QUOTES) . "</td>\n";
	echo "\t\t<td><pre>" . $row[2] . "</pre></td>\n";
    echo "\t</tr>\n";
}
echo "</table><hr>\n";

// Free resultset
pg_free_result($events);

// Closing connection
pg_close($dbconn);
?>
</body>
</html>
